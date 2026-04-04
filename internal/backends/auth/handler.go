package auth

import (
	"encoding/json"
	"net/http"
	"strings"
)

type Handler struct {
	store *Store
}

func NewHandler(jwtSecret string) *Handler {
	return &Handler{store: NewStore(jwtSecret)}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", h.Health)
	mux.HandleFunc("POST /register", h.Register)
	mux.HandleFunc("POST /login", h.Login)
	mux.HandleFunc("POST /logout", h.Logout)
	mux.HandleFunc("POST /refresh", h.Refresh)
	mux.HandleFunc("GET /me", h.authMiddleware(h.Me))
	mux.HandleFunc("PUT /me", h.authMiddleware(h.UpdateMe))
	mux.HandleFunc("POST /change-password", h.authMiddleware(h.ChangePassword))
	mux.HandleFunc("GET /validate", h.Validate)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func (h *Handler) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if len(auth) < 8 || !strings.HasPrefix(strings.ToLower(auth), "bearer ") {
			writeError(w, http.StatusUnauthorized, "missing or invalid authorization header")
			return
		}

		tokenStr := auth[7:]
		claims, err := h.store.ValidateToken(tokenStr)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid token")
			return
		}

		userID := int(claims["user_id"].(float64))
		user, exists := h.store.GetUserByID(userID)
		if !exists {
			writeError(w, http.StatusUnauthorized, "user not found")
			return
		}

		r.Header.Set("X-User-ID", string(rune(userID)))
		r.Header.Set("X-User-Email", user.Email)
		r.Header.Set("X-User-Role", user.Role)

		next(w, r)
	}
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "healthy", "service": "auth-service"})
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Name     string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if input.Email == "" || input.Password == "" || input.Name == "" {
		writeError(w, http.StatusBadRequest, "email, password and name are required")
		return
	}

	user, accessToken, refreshToken, err := h.store.Register(input.Email, input.Password, input.Name)
	if err != nil {
		status := http.StatusBadRequest
		if err == ErrEmailExists {
			status = http.StatusConflict
		}
		writeError(w, status, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"user":          user,
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"expires_in":    900,
	})
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, accessToken, refreshToken, err := h.store.Login(input.Email, input.Password)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"user":          user,
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"expires_in":    900,
	})
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	var input struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	h.store.Logout(input.RefreshToken)
	writeJSON(w, http.StatusOK, map[string]string{"message": "logged out successfully"})
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var input struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	accessToken, refreshToken, err := h.store.Refresh(input.RefreshToken)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid refresh token")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"expires_in":    900,
	})
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	email := r.Header.Get("X-User-Email")
	user, _ := h.store.GetUserByEmail(email)
	writeJSON(w, http.StatusOK, user)
}

func (h *Handler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	email := r.Header.Get("X-User-Email")
	user, _ := h.store.GetUserByEmail(email)

	var input struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	updated, _ := h.store.UpdateUser(user.ID, input.Name)
	writeJSON(w, http.StatusOK, updated)
}

func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	email := r.Header.Get("X-User-Email")
	user, _ := h.store.GetUserByEmail(email)

	var input struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.store.ChangePassword(user.ID, input.OldPassword, input.NewPassword); err != nil {
		status := http.StatusBadRequest
		if err == ErrInvalidCredentials {
			status = http.StatusUnauthorized
		}
		writeError(w, status, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "password changed successfully"})
}

func (h *Handler) Validate(w http.ResponseWriter, r *http.Request) {
	auth := r.Header.Get("Authorization")
	if len(auth) < 8 || !strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		writeError(w, http.StatusUnauthorized, "missing or invalid authorization header")
		return
	}

	tokenStr := auth[7:]
	claims, err := h.store.ValidateToken(tokenStr)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid token")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"valid":   true,
		"user_id": claims["user_id"],
		"email":   claims["email"],
		"role":    claims["role"],
	})
}
