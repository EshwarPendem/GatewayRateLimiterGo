package api

import "errors"

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrProductNotFound   = errors.New("product not found")
	ErrInsufficientStock = errors.New("insufficient stock")
	ErrOrderNotFound     = errors.New("order not found")
	ErrInvalidStatus     = errors.New("invalid status")
)

var ValidOrderStatuses = map[string]bool{
	"pending":    true,
	"processing": true,
	"shipped":    true,
	"completed":  true,
	"cancelled":  true,
}
