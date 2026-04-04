package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/example/go-rate-limiter-gateway/internal/backends/api"
	"github.com/example/go-rate-limiter-gateway/internal/backends/auth"
	"github.com/example/go-rate-limiter-gateway/internal/gateway"
	"github.com/example/go-rate-limiter-gateway/internal/middleware"
	"github.com/example/go-rate-limiter-gateway/internal/ratelimiter"
	"github.com/example/go-rate-limiter-gateway/internal/redis"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Server struct {
		Port         int           `yaml:"port"`
		ReadTimeout  time.Duration `yaml:"read_timeout"`
		WriteTimeout time.Duration `yaml:"write_timeout"`
	} `yaml:"server"`

	Redis struct {
		Host     string `yaml:"host"`
		Port     int    `yaml:"port"`
		Password string `yaml:"password"`
		DB       int    `yaml:"db"`
		PoolSize int    `yaml:"pool_size"`
	} `yaml:"redis"`

	RateLimiter struct {
		Algorithm   string `yaml:"algorithm"`
		TokenBucket struct {
			Rate     int `yaml:"rate"`
			Capacity int `yaml:"capacity"`
		} `yaml:"token_bucket"`
		SlidingWindow struct {
			WindowSize  time.Duration `yaml:"window_size"`
			MaxRequests int           `yaml:"max_requests"`
		} `yaml:"sliding_window"`
	} `yaml:"rate_limiter"`

	JWT struct {
		Secret string        `yaml:"secret"`
		Issuer string        `yaml:"issuer"`
		Expiry time.Duration `yaml:"expiry"`
	} `yaml:"jwt"`

	Backends []struct {
		Name     string `yaml:"name"`
		Prefix   string `yaml:"prefix"`
		Target   string `yaml:"target"`
		Embedded bool   `yaml:"embedded"`
		Port     int    `yaml:"port"`
	} `yaml:"backends"`

	Logging struct {
		Level  string `yaml:"level"`
		Format string `yaml:"format"`
	} `yaml:"logging"`
}

func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "config/config.yaml"
	}

	cfg, err := loadConfig(configPath)
	if err != nil {
		logger.Error("failed to load config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	redisClient, err := redis.NewClient(redis.Config{
		Host:     cfg.Redis.Host,
		Port:     cfg.Redis.Port,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
		PoolSize: cfg.Redis.PoolSize,
	})
	if err != nil {
		logger.Error("failed to connect to redis", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer redisClient.Close()

	var rateLimiter ratelimiter.RateLimiter
	switch cfg.RateLimiter.Algorithm {
	case "sliding_window":
		rateLimiter = ratelimiter.NewSlidingWindow(redisClient, ratelimiter.SlidingWindowConfig{
			WindowSize:  cfg.RateLimiter.SlidingWindow.WindowSize,
			MaxRequests: cfg.RateLimiter.SlidingWindow.MaxRequests,
		})
	default:
		rateLimiter = ratelimiter.NewTokenBucket(redisClient, ratelimiter.TokenBucketConfig{
			Rate:     cfg.RateLimiter.TokenBucket.Rate,
			Capacity: cfg.RateLimiter.TokenBucket.Capacity,
		})
	}

	embeddedServers := startEmbeddedBackends(cfg, logger)

	backends := make([]gateway.Backend, len(cfg.Backends))
	for i, b := range cfg.Backends {
		target := b.Target
		if b.Embedded && b.Port != 0 {
			target = fmt.Sprintf("http://localhost:%d", b.Port)
		}
		backends[i] = gateway.Backend{
			Name:   b.Name,
			Prefix: b.Prefix,
			Target: target,
		}
	}

	proxy, err := gateway.NewProxy(backends, logger)
	if err != nil {
		logger.Error("failed to create proxy", slog.String("error", err.Error()))
		os.Exit(1)
	}

	loggingMiddleware := middleware.NewLoggingMiddleware(logger)
	rateLimitMiddleware := middleware.NewRateLimitMiddleware(rateLimiter, nil)
	authMiddleware := middleware.NewAuthMiddleware(middleware.JWTConfig{
		Secret: cfg.JWT.Secret,
		Issuer: cfg.JWT.Issuer,
		Expiry: cfg.JWT.Expiry,
	})

	mux := http.NewServeMux()

	mux.HandleFunc("/health", proxy.HealthCheck)

	mux.HandleFunc("/gateway/token", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		userID := r.FormValue("user_id")
		if userID == "" {
			userID = "anonymous"
		}

		token, err := authMiddleware.GenerateToken(userID, []string{"user"})
		if err != nil {
			http.Error(w, "Failed to generate token", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"token":"` + token + `"}`))
	})

	protectedHandler := loggingMiddleware.Handler(
		rateLimitMiddleware.Handler(
			authMiddleware.Handler(proxy),
		),
	)

	publicHandler := loggingMiddleware.Handler(
		rateLimitMiddleware.Handler(proxy),
	)

	mux.Handle("/api/", protectedHandler)
	mux.Handle("/", publicHandler)

	addr := ":8080"
	if cfg.Server.Port != 0 {
		addr = fmt.Sprintf(":%d", cfg.Server.Port)
	}

	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	go func() {
		logger.Info("starting gateway server", slog.String("addr", server.Addr))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down servers...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for _, s := range embeddedServers {
		if err := s.Shutdown(ctx); err != nil {
			logger.Error("embedded server shutdown error", slog.String("error", err.Error()))
		}
	}

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("gateway server shutdown error", slog.String("error", err.Error()))
	}

	logger.Info("all servers stopped")
}

func startEmbeddedBackends(cfg *Config, logger *slog.Logger) []*http.Server {
	var servers []*http.Server

	for _, b := range cfg.Backends {
		if !b.Embedded || b.Port == 0 {
			continue
		}

		switch b.Name {
		case "api-service":
			server := startAPIService(b.Port, logger)
			servers = append(servers, server)

		case "auth-service":
			server := startAuthService(b.Port, cfg.JWT.Secret, logger)
			servers = append(servers, server)
		}
	}

	return servers
}

func startAPIService(port int, logger *slog.Logger) *http.Server {
	mux := http.NewServeMux()
	handler := api.NewHandler()
	handler.RegisterRoutes(mux)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	go func() {
		logger.Info("starting embedded api-service", slog.Int("port", port))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("api-service error", slog.String("error", err.Error()))
		}
	}()

	return server
}

func startAuthService(port int, jwtSecret string, logger *slog.Logger) *http.Server {
	mux := http.NewServeMux()
	handler := auth.NewHandler(jwtSecret)
	handler.RegisterRoutes(mux)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	go func() {
		logger.Info("starting embedded auth-service", slog.Int("port", port))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("auth-service error", slog.String("error", err.Error()))
		}
	}()

	return server
}
