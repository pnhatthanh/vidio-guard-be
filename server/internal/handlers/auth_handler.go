package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pnhatthanh/vidio-guard-be/internal/apperror"
	"github.com/pnhatthanh/vidio-guard-be/internal/dto"
	"github.com/pnhatthanh/vidio-guard-be/internal/services"
)

type AuthHandler interface {
	Register() gin.HandlerFunc
	Login() gin.HandlerFunc
	LoginWithGoogle() gin.HandlerFunc
	RefreshToken() gin.HandlerFunc
	Logout() gin.HandlerFunc
}

type authHandler struct {
	auth services.AuthService
}

func NewAuthHandler(auth services.AuthService) AuthHandler {
	return &authHandler{auth: auth}
}

func (h *authHandler) Register() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req dto.RegisterRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.Error(apperror.NewBadRequestError(err.Error()))
			return
		}

		user, err := h.auth.Register(c.Request.Context(), req)
		if err != nil {
			c.Error(err)
			return
		}
		c.JSON(http.StatusCreated, user)
	}
}

func (h *authHandler) Login() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req dto.LoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.Error(apperror.NewBadRequestError(err.Error()))
			return
		}

		res, err := h.auth.Login(c.Request.Context(), req)
		if err != nil {
			c.Error(err)
			return
		}
		c.JSON(http.StatusOK, res)
	}
}

func (h *authHandler) LoginWithGoogle() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req dto.GoogleLoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.Error(apperror.NewBadRequestError(err.Error()))
			return
		}

		res, err := h.auth.LoginWithGoogle(c.Request.Context(), req.IDToken)
		if err != nil {
			c.Error(err)
			return
		}
		c.JSON(http.StatusOK, res)
	}
}

func (h *authHandler) RefreshToken() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req dto.RefreshRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.Error(apperror.NewBadRequestError(err.Error()))
			return
		}

		res, err := h.auth.RefreshToken(c.Request.Context(), req.RefreshToken)
		if err != nil {
			c.Error(err)
			return
		}
		c.JSON(http.StatusOK, res)
	}
}

func (h *authHandler) Logout() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req dto.LogoutRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.Error(apperror.NewBadRequestError(err.Error()))
			return
		}

		userID, ok := c.Get("userID")
		if !ok {
			c.Error(apperror.NewUnauthorizedError("unauthorized"))
			return
		}
		jti, ok := c.Get("jti")
		if !ok {
			c.Error(apperror.NewUnauthorizedError("unauthorized"))
			return
		}
		expiresAt, ok := c.Get("expiresAt")
		if !ok {
			c.Error(apperror.NewUnauthorizedError("unauthorized"))
			return
		}

		if err := h.auth.Logout(
			c.Request.Context(),
			jti.(string),
			userID.(uuid.UUID),
			expiresAt.(time.Time),
			req.RefreshToken,
		); err != nil {
			c.Error(err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "logged out"})
	}
}
