package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tirenn/commerce/backend/internal/domain"
	"github.com/tirenn/commerce/backend/internal/response"
)

type Handler struct {
	useCase UseCase
}

func NewHandler(useCase UseCase) *Handler {
	return &Handler{useCase: useCase}
}

// RegisterRoutes sets up all authentication endpoints
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	authGroup := rg.Group("/auth")
	{
		authGroup.POST("/register", h.Register)
		authGroup.POST("/login", h.Login)
		authGroup.GET("/me", authMiddleware, h.Me)
	}
}

// Register handles user registration
func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "Invalid registration input", domain.ErrBadRequest)
		return
	}

	res, err := h.useCase.Register(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, err.Error(), err)
		return
	}

	response.Created(c, "User registered successfully", res)
}

// Login handles user authentication
func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "Invalid login credentials", domain.ErrBadRequest)
		return
	}

	res, err := h.useCase.Login(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, err.Error(), err)
		return
	}

	response.Success(c, http.StatusOK, "Login successful", res)
}

// Me returns currently authenticated user profile
func (h *Handler) Me(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		response.Error(c, "Authentication required", domain.ErrUnauthorized)
		return
	}

	userID, ok := userIDVal.(uint)
	if !ok {
		response.Error(c, "Invalid user token", domain.ErrUnauthorized)
		return
	}

	profile, err := h.useCase.GetProfile(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, "User not found", err)
		return
	}

	response.Success(c, http.StatusOK, "Profile retrieved", profile)
}
