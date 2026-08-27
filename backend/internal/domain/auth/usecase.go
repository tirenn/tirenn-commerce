package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gocommerce-backend/internal/config"
	"gocommerce-backend/internal/logger"
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
		errExists := errors.New("email already registered")
		logger.Warn(ctx, "usecase", fmt.Sprintf("registration failed: %s is already taken", email), errExists)
		return nil, errExists
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		logger.Error(ctx, "usecase", "database error during email check", err)
		return nil, err
	}

	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		logger.Error(ctx, "usecase", "failed to hash password", err)
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
		logger.Error(ctx, "usecase", "failed to create user in database", err)
		return nil, err
	}

	token, err := utils.GenerateJWT(user.ID, user.Email, user.Role, user.Name, u.config.JWTSecret, u.config.JWTExpireHours)
	if err != nil {
		logger.Error(ctx, "usecase", "failed to generate jwt token after registration", err)
		return nil, errors.New("failed to generate token")
	}

	logger.Info(ctx, "usecase", fmt.Sprintf("user registered successfully: %s (role: %s)", user.Email, user.Role))

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
			logger.Warn(ctx, "usecase", fmt.Sprintf("login attempt for non-existent email: %s", email), err)
			return nil, errors.New("invalid email or password")
		}
		logger.Error(ctx, "usecase", "database error during login lookup", err)
		return nil, err
	}

	if user.Status == StatusSuspended {
		errSuspended := errors.New("your account has been suspended, please contact support")
		logger.Warn(ctx, "usecase", fmt.Sprintf("login attempt by suspended user %s", email), errSuspended)
		return nil, errSuspended
	}

	if !utils.CheckPasswordHash(req.Password, user.PasswordHash) {
		errInvalid := errors.New("invalid email or password")
		logger.Warn(ctx, "usecase", fmt.Sprintf("invalid password entered for %s", email), errInvalid)
		return nil, errInvalid
	}

	token, err := utils.GenerateJWT(user.ID, user.Email, user.Role, user.Name, u.config.JWTSecret, u.config.JWTExpireHours)
	if err != nil {
		logger.Error(ctx, "usecase", "failed to generate jwt token during login", err)
		return nil, errors.New("failed to generate authentication token")
	}

	logger.Info(ctx, "usecase", fmt.Sprintf("user logged in successfully: %s", user.Email))

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
		logger.Warn(ctx, "usecase", fmt.Sprintf("profile not found for user ID %d", userID), err)
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
