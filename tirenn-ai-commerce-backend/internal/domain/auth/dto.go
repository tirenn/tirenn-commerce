package auth

import "time"

// RegisterRequest defines the input schema for user registration
type RegisterRequest struct {
	Name     string `json:"name" binding:"required,min=2,max=100"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	Phone    string `json:"phone"`
	Address  string `json:"address"`
	Role     string `json:"role"` // Optional, defaults to CUSTOMER
}

// LoginRequest defines the input schema for user authentication
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// UserResponse represents safe user profile presentation data
type UserResponse struct {
	ID        uint      `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	Phone     string    `json:"phone"`
	Address   string    `json:"address"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// AuthResponse represents authentication success output with JWT token
type AuthResponse struct {
	Token string       `json:"token"`
	User  UserResponse `json:"user"`
}
