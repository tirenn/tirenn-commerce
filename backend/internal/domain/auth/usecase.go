package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/tirenn/commerce/backend/internal/config"
	"github.com/tirenn/commerce/backend/internal/domain"
	"github.com/tirenn/commerce/backend/internal/logger"
	"github.com/tirenn/commerce/backend/internal/security"
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

func (u *useCase) Register(ctx context.Context, req *RegisterRequest) (authResp *AuthResponse, err error) {
	defer logger.Track(ctx, "usecase.auth", "Register")(&err, map[string]interface{}{"email": req.Email})
	email := strings.ToLower(strings.TrimSpace(req.Email))

	// Check existing user
	existing, err := u.repo.FindByEmail(ctx, email)
	if err == nil && existing != nil {
		logger.Warn(ctx, "usecase", fmt.Sprintf("registration failed: %s is already taken", email), domain.ErrEmailAlreadyExists)
		return nil, domain.ErrEmailAlreadyExists
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		logger.Error(ctx, "usecase", "database error during email check", err)
		return nil, err
	}

	hashedPassword, err := security.HashPassword(req.Password)
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

	token, err := security.GenerateJWT(
		user.ID,
		user.Email,
		user.Role,
		user.Name,
		u.config.JWTSecret,
		u.config.JWTExpireHours,
	)
	if err != nil {
		logger.Error(ctx, "usecase", "failed to generate jwt token", err)
		return nil, err
	}

	return &AuthResponse{
		Token: token,
		User:  user.ToResponse(),
	}, nil
}

func (u *useCase) Login(ctx context.Context, req *LoginRequest) (authResp *AuthResponse, err error) {
	defer logger.Track(ctx, "usecase.auth", "Login")(&err, map[string]interface{}{"email": req.Email})
	email := strings.ToLower(strings.TrimSpace(req.Email))

	user, err := u.repo.FindByEmail(ctx, email)
	if err != nil {
		logger.Warn(ctx, "usecase", fmt.Sprintf("login failed: user with email %s not found", email), domain.ErrInvalidCredentials)
		return nil, domain.ErrInvalidCredentials
	}

	if user.Status == StatusSuspended {
		logger.Warn(ctx, "usecase", fmt.Sprintf("login rejected: account %s is suspended", email), domain.ErrForbidden)
		return nil, domain.ErrForbidden
	}

	if !security.CheckPasswordHash(req.Password, user.PasswordHash) {
		logger.Warn(ctx, "usecase", fmt.Sprintf("login failed: invalid password for %s", email), domain.ErrInvalidCredentials)
		return nil, domain.ErrInvalidCredentials
	}

	token, err := security.GenerateJWT(
		user.ID,
		user.Email,
		user.Role,
		user.Name,
		u.config.JWTSecret,
		u.config.JWTExpireHours,
	)
	if err != nil {
		logger.Error(ctx, "usecase", "failed to generate jwt token for login", err)
		return nil, err
	}

	return &AuthResponse{
		Token: token,
		User:  user.ToResponse(),
	}, nil
}

func (u *useCase) GetProfile(ctx context.Context, userID uint) (*UserResponse, error) {
	user, err := u.repo.FindByID(ctx, userID)
	if err != nil {
		logger.Warn(ctx, "usecase", fmt.Sprintf("failed to get profile for userID %d", userID), domain.ErrNotFound)
		return nil, domain.ErrNotFound
	}

	resp := user.ToResponse()
	return &resp, nil
}
