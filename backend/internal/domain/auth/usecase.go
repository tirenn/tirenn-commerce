package auth

import (
	"context"
	"errors"
	"strings"

	"gocommerce-backend/internal/config"
	"gocommerce-backend/internal/utils"
	"gorm.io/gorm"
)

type UseCase interface {
	Register(ctx context.Context, req *RegisterRequest) (*AuthResponse, error)
	Login(ctx context.Context, req *LoginRequest) (*AuthResponse, error)
	GetProfile(ctx context.Context, userID uint) (*UserResponse, error)
}

type useCase struct {
	repo   Repository
	config *config.Config
}

func NewUseCase(repo Repository, cfg *config.Config) UseCase {
	return &useCase{
		repo:   repo,
		config: cfg,
	}
}

func (u *useCase) Register(ctx context.Context, req *RegisterRequest) (*AuthResponse, error) {
	email := strings.ToLower(strings.TrimSpace(req.Email))

	// Check existing user
	existing, err := u.repo.FindByEmail(ctx, email)
	if err == nil && existing != nil {
		return nil, errors.New("email already registered")
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		return nil, errors.New("failed to hash password")
	}

	role := RoleCustomer
	if req.Role == RoleAdmin {
		role = RoleAdmin
	}

	user := &User{
		Name:         req.Name,
		Email:        email,
		PasswordHash: hashedPassword,
		Role:         role,
		Phone:        req.Phone,
		Address:      req.Address,
		Status:       StatusActive,
	}

	if err := u.repo.Create(ctx, user); err != nil {
		return nil, err
	}

	token, err := utils.GenerateJWT(user.ID, user.Email, user.Role, user.Name, u.config.JWTSecret, u.config.JWTExpireHours)
	if err != nil {
		return nil, errors.New("failed to generate token")
	}

	return &AuthResponse{
		Token: token,
		User: UserResponse{
			ID:        user.ID,
			Name:      user.Name,
			Email:     user.Email,
			Role:      user.Role,
			Phone:     user.Phone,
			Address:   user.Address,
			Status:    user.Status,
			CreatedAt: user.CreatedAt,
		},
	}, nil
}

func (u *useCase) Login(ctx context.Context, req *LoginRequest) (*AuthResponse, error) {
	email := strings.ToLower(strings.TrimSpace(req.Email))

	user, err := u.repo.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("invalid email or password")
		}
		return nil, err
	}

	if user.Status == StatusSuspended {
		return nil, errors.New("your account has been suspended, please contact support")
	}

	if !utils.CheckPasswordHash(req.Password, user.PasswordHash) {
		return nil, errors.New("invalid email or password")
	}

	token, err := utils.GenerateJWT(user.ID, user.Email, user.Role, user.Name, u.config.JWTSecret, u.config.JWTExpireHours)
	if err != nil {
		return nil, errors.New("failed to generate authentication token")
	}

	return &AuthResponse{
		Token: token,
		User: UserResponse{
			ID:        user.ID,
			Name:      user.Name,
			Email:     user.Email,
			Role:      user.Role,
			Phone:     user.Phone,
			Address:   user.Address,
			Status:    user.Status,
			CreatedAt: user.CreatedAt,
		},
	}, nil
}

func (u *useCase) GetProfile(ctx context.Context, userID uint) (*UserResponse, error) {
	user, err := u.repo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &UserResponse{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		Role:      user.Role,
		Phone:     user.Phone,
		Address:   user.Address,
		Status:    user.Status,
		CreatedAt: user.CreatedAt,
	}, nil
}
