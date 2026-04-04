package auth

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID           int       `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Name         string    `json:"name"`
	Role         string    `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
}

type Store struct {
	mu            sync.RWMutex
	users         map[int]User
	usersByEmail  map[string]int
	refreshTokens map[string]int
	nextUserID    int
	jwtSecret     []byte
}

func NewStore(jwtSecret string) *Store {
	s := &Store{
		users:         make(map[int]User),
		usersByEmail:  make(map[string]int),
		refreshTokens: make(map[string]int),
		nextUserID:    1,
		jwtSecret:     []byte(jwtSecret),
	}
	s.seedData()
	return s
}

func (s *Store) seedData() {
	hash, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	s.users[1] = User{ID: 1, Email: "admin@example.com", PasswordHash: string(hash), Name: "Admin", Role: "admin", CreatedAt: time.Now()}
	s.usersByEmail["admin@example.com"] = 1

	hash2, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	s.users[2] = User{ID: 2, Email: "user@example.com", PasswordHash: string(hash2), Name: "User", Role: "user", CreatedAt: time.Now()}
	s.usersByEmail["user@example.com"] = 2

	s.nextUserID = 3
}

func (s *Store) Register(email, password, name string) (User, string, string, error) {
	if len(password) < 8 {
		return User{}, "", "", ErrPasswordTooShort
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.usersByEmail[email]; exists {
		return User{}, "", "", ErrEmailExists
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return User{}, "", "", err
	}

	user := User{
		ID:           s.nextUserID,
		Email:        email,
		PasswordHash: string(hash),
		Name:         name,
		Role:         "user",
		CreatedAt:    time.Now(),
	}
	s.users[user.ID] = user
	s.usersByEmail[email] = user.ID
	s.nextUserID++

	accessToken, _ := s.generateJWT(user)
	refreshToken := generateToken(32)
	s.refreshTokens[refreshToken] = user.ID

	return user, accessToken, refreshToken, nil
}

func (s *Store) Login(email, password string) (User, string, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	userID, exists := s.usersByEmail[email]
	if !exists {
		return User{}, "", "", ErrInvalidCredentials
	}
	user := s.users[userID]

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return User{}, "", "", ErrInvalidCredentials
	}

	accessToken, _ := s.generateJWT(user)
	refreshToken := generateToken(32)
	s.refreshTokens[refreshToken] = user.ID

	return user, accessToken, refreshToken, nil
}

func (s *Store) Logout(refreshToken string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.refreshTokens, refreshToken)
}

func (s *Store) Refresh(refreshToken string) (string, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	userID, exists := s.refreshTokens[refreshToken]
	if !exists {
		return "", "", ErrInvalidRefreshToken
	}

	delete(s.refreshTokens, refreshToken)

	user := s.users[userID]
	accessToken, _ := s.generateJWT(user)
	newRefreshToken := generateToken(32)
	s.refreshTokens[newRefreshToken] = userID

	return accessToken, newRefreshToken, nil
}

func (s *Store) GetUserByEmail(email string) (User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	userID, exists := s.usersByEmail[email]
	if !exists {
		return User{}, false
	}
	return s.users[userID], true
}

func (s *Store) GetUserByID(id int) (User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	user, exists := s.users[id]
	return user, exists
}

func (s *Store) UpdateUser(id int, name string) (User, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	user, exists := s.users[id]
	if !exists {
		return User{}, false
	}
	user.Name = name
	s.users[id] = user
	return user, true
}

func (s *Store) ChangePassword(id int, oldPassword, newPassword string) error {
	if len(newPassword) < 8 {
		return ErrPasswordTooShort
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	user, exists := s.users[id]
	if !exists {
		return ErrUserNotFound
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(oldPassword)); err != nil {
		return ErrInvalidCredentials
	}

	hash, _ := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	user.PasswordHash = string(hash)
	s.users[id] = user

	return nil
}

func (s *Store) ValidateToken(tokenStr string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		return s.jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

func (s *Store) generateJWT(user User) (string, error) {
	claims := jwt.MapClaims{
		"user_id": user.ID,
		"email":   user.Email,
		"role":    user.Role,
		"exp":     time.Now().Add(15 * time.Minute).Unix(),
		"iat":     time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

func generateToken(length int) string {
	b := make([]byte, length)
	rand.Read(b)
	return hex.EncodeToString(b)
}
