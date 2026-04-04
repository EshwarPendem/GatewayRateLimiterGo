package gateway

import (
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

type Backend struct {
	Name   string
	Prefix string
	Target string
}

type Proxy struct {
	backends map[string]*httputil.ReverseProxy
	routes   []Backend
	logger   *slog.Logger
}

func NewProxy(backends []Backend, logger *slog.Logger) (*Proxy, error) {
	if logger == nil {
		logger = slog.Default()
	}

	p := &Proxy{
		backends: make(map[string]*httputil.ReverseProxy),
		routes:   backends,
		logger:   logger,
	}

	for _, b := range backends {
		targetURL, err := url.Parse(b.Target)
		if err != nil {
			return nil, err
		}

		proxy := httputil.NewSingleHostReverseProxy(targetURL)
		proxy.ErrorHandler = p.errorHandler(b.Name)
		proxy.ModifyResponse = p.modifyResponse(b.Name)

		p.backends[b.Prefix] = proxy
	}

	return p, nil
}

func (p *Proxy) errorHandler(backendName string) func(http.ResponseWriter, *http.Request, error) {
	return func(w http.ResponseWriter, r *http.Request, err error) {
		p.logger.Error("proxy error",
			slog.String("backend", backendName),
			slog.String("path", r.URL.Path),
			slog.String("error", err.Error()),
		)
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
	}
}

func (p *Proxy) modifyResponse(backendName string) func(*http.Response) error {
	return func(resp *http.Response) error {
		resp.Header.Set("X-Proxy-By", "go-rate-limiter-gateway")
		resp.Header.Set("X-Backend", backendName)
		return nil
	}
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for _, route := range p.routes {
		if strings.HasPrefix(r.URL.Path, route.Prefix) {
			proxy := p.backends[route.Prefix]
			if proxy != nil {
				r.URL.Path = strings.TrimPrefix(r.URL.Path, route.Prefix)
				if r.URL.Path == "" {
					r.URL.Path = "/"
				}
				proxy.ServeHTTP(w, r)
				return
			}
		}
	}

	http.Error(w, "Not Found", http.StatusNotFound)
}

func (p *Proxy) HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"healthy"}`))
}
