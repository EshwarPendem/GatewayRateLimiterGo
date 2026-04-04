package api

import (
	"sync"
	"time"
)

type User struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

type Product struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	Stock       int     `json:"stock"`
}

type Order struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	ProductID int       `json:"product_id"`
	Quantity  int       `json:"quantity"`
	Total     float64   `json:"total"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type Store struct {
	mu       sync.RWMutex
	users    map[int]User
	products map[int]Product
	orders   map[int]Order
	nextID   map[string]int
}

func NewStore() *Store {
	s := &Store{
		users:    make(map[int]User),
		products: make(map[int]Product),
		orders:   make(map[int]Order),
		nextID:   map[string]int{"users": 1, "products": 1, "orders": 1},
	}
	s.seedData()
	return s
}

func (s *Store) seedData() {
	s.users[1] = User{ID: 1, Name: "John Doe", Email: "john@example.com", CreatedAt: time.Now()}
	s.users[2] = User{ID: 2, Name: "Jane Smith", Email: "jane@example.com", CreatedAt: time.Now()}
	s.nextID["users"] = 3

	s.products[1] = Product{ID: 1, Name: "Laptop", Description: "High-performance laptop", Price: 999.99, Stock: 50}
	s.products[2] = Product{ID: 2, Name: "Mouse", Description: "Wireless mouse", Price: 29.99, Stock: 200}
	s.products[3] = Product{ID: 3, Name: "Keyboard", Description: "Mechanical keyboard", Price: 79.99, Stock: 100}
	s.nextID["products"] = 4

	s.orders[1] = Order{ID: 1, UserID: 1, ProductID: 1, Quantity: 1, Total: 999.99, Status: "completed", CreatedAt: time.Now()}
	s.nextID["orders"] = 2
}

func (s *Store) ListUsers() []User {
	s.mu.RLock()
	defer s.mu.RUnlock()
	users := make([]User, 0, len(s.users))
	for _, u := range s.users {
		users = append(users, u)
	}
	return users
}

func (s *Store) GetUser(id int) (User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.users[id]
	return u, ok
}

func (s *Store) CreateUser(name, email string) User {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := s.nextID["users"]
	s.nextID["users"]++
	user := User{ID: id, Name: name, Email: email, CreatedAt: time.Now()}
	s.users[id] = user
	return user
}

func (s *Store) UpdateUser(id int, name, email string) (User, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	user, exists := s.users[id]
	if !exists {
		return User{}, false
	}
	user.Name = name
	user.Email = email
	s.users[id] = user
	return user, true
}

func (s *Store) DeleteUser(id int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.users[id]; !exists {
		return false
	}
	delete(s.users, id)
	return true
}

func (s *Store) ListProducts() []Product {
	s.mu.RLock()
	defer s.mu.RUnlock()
	products := make([]Product, 0, len(s.products))
	for _, p := range s.products {
		products = append(products, p)
	}
	return products
}

func (s *Store) GetProduct(id int) (Product, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.products[id]
	return p, ok
}

func (s *Store) CreateProduct(name, description string, price float64, stock int) Product {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := s.nextID["products"]
	s.nextID["products"]++
	product := Product{ID: id, Name: name, Description: description, Price: price, Stock: stock}
	s.products[id] = product
	return product
}

func (s *Store) UpdateProduct(id int, name, description string, price float64, stock int) (Product, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	product, exists := s.products[id]
	if !exists {
		return Product{}, false
	}
	product.Name = name
	product.Description = description
	product.Price = price
	product.Stock = stock
	s.products[id] = product
	return product, true
}

func (s *Store) DeleteProduct(id int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.products[id]; !exists {
		return false
	}
	delete(s.products, id)
	return true
}

func (s *Store) ListOrders() []Order {
	s.mu.RLock()
	defer s.mu.RUnlock()
	orders := make([]Order, 0, len(s.orders))
	for _, o := range s.orders {
		orders = append(orders, o)
	}
	return orders
}

func (s *Store) GetOrder(id int) (Order, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	o, ok := s.orders[id]
	return o, ok
}

func (s *Store) CreateOrder(userID, productID, quantity int) (Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[userID]; !exists {
		return Order{}, ErrUserNotFound
	}

	product, exists := s.products[productID]
	if !exists {
		return Order{}, ErrProductNotFound
	}

	if product.Stock < quantity {
		return Order{}, ErrInsufficientStock
	}

	product.Stock -= quantity
	s.products[productID] = product

	id := s.nextID["orders"]
	s.nextID["orders"]++
	order := Order{
		ID:        id,
		UserID:    userID,
		ProductID: productID,
		Quantity:  quantity,
		Total:     product.Price * float64(quantity),
		Status:    "pending",
		CreatedAt: time.Now(),
	}
	s.orders[id] = order
	return order, nil
}

func (s *Store) UpdateOrderStatus(id int, status string) (Order, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	order, exists := s.orders[id]
	if !exists {
		return Order{}, false
	}
	order.Status = status
	s.orders[id] = order
	return order, true
}
