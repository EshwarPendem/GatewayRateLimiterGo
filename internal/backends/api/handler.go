package api

import (
	"encoding/json"
	"net/http"
	"strconv"
)

type Handler struct {
	store *Store
}

func NewHandler() *Handler {
	return &Handler{store: NewStore()}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", h.Health)
	mux.HandleFunc("GET /users", h.ListUsers)
	mux.HandleFunc("GET /users/{id}", h.GetUser)
	mux.HandleFunc("POST /users", h.CreateUser)
	mux.HandleFunc("PUT /users/{id}", h.UpdateUser)
	mux.HandleFunc("DELETE /users/{id}", h.DeleteUser)

	mux.HandleFunc("GET /products", h.ListProducts)
	mux.HandleFunc("GET /products/{id}", h.GetProduct)
	mux.HandleFunc("POST /products", h.CreateProduct)
	mux.HandleFunc("PUT /products/{id}", h.UpdateProduct)
	mux.HandleFunc("DELETE /products/{id}", h.DeleteProduct)

	mux.HandleFunc("GET /orders", h.ListOrders)
	mux.HandleFunc("GET /orders/{id}", h.GetOrder)
	mux.HandleFunc("POST /orders", h.CreateOrder)
	mux.HandleFunc("PUT /orders/{id}/status", h.UpdateOrderStatus)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "healthy", "service": "api-service"})
}

func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.store.ListUsers())
}

func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user id")
		return
	}
	user, ok := h.store.GetUser(id)
	if !ok {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	user := h.store.CreateUser(input.Name, input.Email)
	writeJSON(w, http.StatusCreated, user)
}

func (h *Handler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user id")
		return
	}
	var input struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	user, ok := h.store.UpdateUser(id, input.Name, input.Email)
	if !ok {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func (h *Handler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user id")
		return
	}
	if !h.store.DeleteUser(id) {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ListProducts(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.store.ListProducts())
}

func (h *Handler) GetProduct(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid product id")
		return
	}
	product, ok := h.store.GetProduct(id)
	if !ok {
		writeError(w, http.StatusNotFound, "product not found")
		return
	}
	writeJSON(w, http.StatusOK, product)
}

func (h *Handler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name        string  `json:"name"`
		Description string  `json:"description"`
		Price       float64 `json:"price"`
		Stock       int     `json:"stock"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	product := h.store.CreateProduct(input.Name, input.Description, input.Price, input.Stock)
	writeJSON(w, http.StatusCreated, product)
}

func (h *Handler) UpdateProduct(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid product id")
		return
	}
	var input struct {
		Name        string  `json:"name"`
		Description string  `json:"description"`
		Price       float64 `json:"price"`
		Stock       int     `json:"stock"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	product, ok := h.store.UpdateProduct(id, input.Name, input.Description, input.Price, input.Stock)
	if !ok {
		writeError(w, http.StatusNotFound, "product not found")
		return
	}
	writeJSON(w, http.StatusOK, product)
}

func (h *Handler) DeleteProduct(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid product id")
		return
	}
	if !h.store.DeleteProduct(id) {
		writeError(w, http.StatusNotFound, "product not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ListOrders(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.store.ListOrders())
}

func (h *Handler) GetOrder(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid order id")
		return
	}
	order, ok := h.store.GetOrder(id)
	if !ok {
		writeError(w, http.StatusNotFound, "order not found")
		return
	}
	writeJSON(w, http.StatusOK, order)
}

func (h *Handler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	var input struct {
		UserID    int `json:"user_id"`
		ProductID int `json:"product_id"`
		Quantity  int `json:"quantity"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	order, err := h.store.CreateOrder(input.UserID, input.ProductID, input.Quantity)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, order)
}

func (h *Handler) UpdateOrderStatus(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid order id")
		return
	}
	var input struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if !ValidOrderStatuses[input.Status] {
		writeError(w, http.StatusBadRequest, "invalid status")
		return
	}
	order, ok := h.store.UpdateOrderStatus(id, input.Status)
	if !ok {
		writeError(w, http.StatusNotFound, "order not found")
		return
	}
	writeJSON(w, http.StatusOK, order)
}
