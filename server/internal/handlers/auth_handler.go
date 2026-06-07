package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pnhatthanh/vidio-guard-be/internal/apperror"
	"github.com/pnhatthanh/vidio-guard-be/internal/dto"
	"github.com/pnhatthanh/vidio-guard-be/internal/services"
	"github.com/pnhatthanh/vidio-guard-be/internal/utils"
)

type AuthHandler interface {
	Register() gin.HandlerFunc
	Login() gin.HandlerFunc
	LoginWithGoogle() gin.HandlerFunc
	RefreshToken() gin.HandlerFunc
	Logout() gin.HandlerFunc
	ForgotPassword() gin.HandlerFunc
	ResetPassword() gin.HandlerFunc
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

		userID, err := utils.GetCurrentUserID(c)
		if err != nil {
			c.Error(apperror.NewUnauthorizedError("unauthorized"))
			return
		}
		jti, err := utils.GetJTI(c)
		if err != nil {
			c.Error(apperror.NewUnauthorizedError("unauthorized"))
			return
		}
		expiresAt, err := utils.GetExpiresAt(c)
		if err != nil {
			c.Error(apperror.NewUnauthorizedError("unauthorized"))
			return
		}

		if err := h.auth.Logout(
			c.Request.Context(),
			jti,
			userID,
			expiresAt,
			req.RefreshToken,
		); err != nil {
			c.Error(err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "logged out"})
	}
}

func (h *authHandler) ForgotPassword() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req dto.ForgotPasswordRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.Error(apperror.NewBadRequestError(err.Error()))
			return
		}

		res, err := h.auth.ForgotPassword(c.Request.Context(), req)
		if err != nil {
			c.Error(err)
			return
		}
		c.JSON(http.StatusOK, res)
	}
}

func (h *authHandler) ResetPassword() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req dto.ResetPasswordRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.Error(apperror.NewBadRequestError(err.Error()))
			return
		}

		res, err := h.auth.ResetPassword(c.Request.Context(), req)
		if err != nil {
			c.Error(err)
			return
		}
		c.JSON(http.StatusOK, res)
	}
}
