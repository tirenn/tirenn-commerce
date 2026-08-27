package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gocommerce-backend/internal/utils"
)

type Handler struct {
	useCase UseCase
}

func NewHandler(useCase UseCase) *Handler {
	return &Handler{useCase: useCase}
}

// Register handles user registration
func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "Invalid registration input", err.Error())
		return
	}

	res, err := h.useCase.Register(c.Request.Context(), &req)
	if err != nil {
		utils.BadRequest(c, err.Error(), nil)
		return
	}

	utils.Success(c, http.StatusCreated, "User registered successfully", res)
}

// Login handles user authentication
func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "Invalid login credentials", err.Error())
		return
	}

	res, err := h.useCase.Login(c.Request.Context(), &req)
	if err != nil {
		utils.Unauthorized(c, err.Error())
		return
	}

	utils.Success(c, http.StatusOK, "Login successful", res)
}

// Me returns currently authenticated user profile
func (h *Handler) Me(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		utils.Unauthorized(c, "Authentication required")
		return
	}

	userID, ok := userIDVal.(uint)
	if !ok {
		utils.Unauthorized(c, "Invalid user token")
		return
	}

	profile, err := h.useCase.GetProfile(c.Request.Context(), userID)
	if err != nil {
		utils.NotFound(c, "User not found")
		return
	}

	utils.Success(c, http.StatusOK, "Profile retrieved", profile)
}
