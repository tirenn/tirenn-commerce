package auth

import (
	"time"
)

const (
	RoleAdmin    = "ADMIN"
	RoleCustomer = "CUSTOMER"

	StatusActive    = "ACTIVE"
	StatusSuspended = "SUSPENDED"
)

// User represents the persistent user account domain model
type User struct {
	ID           uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Name         string    `gorm:"size:100;not null" json:"name"`
	Email        string    `gorm:"size:191;unique;not null" json:"email"`
	PasswordHash string    `gorm:"size:255;not null" json:"-"`
	Role         string    `gorm:"size:20;default:'CUSTOMER';not null" json:"role"`
	Phone        string    `gorm:"size:30" json:"phone"`
	Address      string    `gorm:"type:text" json:"address"`
	Status       string    `gorm:"size:20;default:'ACTIVE';not null" json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (User) TableName() string {
	return "users"
}

// ToResponse converts User entity into UserResponse DTO
func (u *User) ToResponse() UserResponse {
	return UserResponse{
		ID:        u.ID,
		Name:      u.Name,
		Email:      u.Email,
		Role:      u.Role,
		Phone:     u.Phone,
		Address:   u.Address,
		Status:    u.Status,
		CreatedAt: u.CreatedAt,
	}
}
