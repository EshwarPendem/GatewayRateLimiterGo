package auth

import "errors"

var (
	ErrEmailExists         = errors.New("email already registered")
	ErrPasswordTooShort    = errors.New("password must be at least 8 characters")
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
	ErrInvalidToken        = errors.New("invalid token")
	ErrUserNotFound        = errors.New("user not found")
	ErrUnauthorized        = errors.New("unauthorized")
)
