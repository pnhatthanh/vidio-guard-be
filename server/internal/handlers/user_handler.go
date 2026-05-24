package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pnhatthanh/vidio-guard-be/internal/apperror"
	"github.com/pnhatthanh/vidio-guard-be/internal/dto"
	"github.com/pnhatthanh/vidio-guard-be/internal/services"
	"github.com/pnhatthanh/vidio-guard-be/internal/utils"
)

type UserHandler interface {
	GetMe() gin.HandlerFunc
	UpdateMe() gin.HandlerFunc
	ChangePassword() gin.HandlerFunc
}

type userHandler struct {
	users services.UserService
}

func NewUserHandler(users services.UserService) UserHandler {
	return &userHandler{users: users}
}

func (h *userHandler) GetMe() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := utils.GetCurrentUserID(c)
		if err != nil {
			c.Error(err)
			return
		}

		res, err := h.users.GetProfile(c.Request.Context(), userID)
		if err != nil {
			c.Error(err)
			return
		}
		c.JSON(http.StatusOK, res)
	}
}

func (h *userHandler) UpdateMe() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := utils.GetCurrentUserID(c)
		if err != nil {
			c.Error(err)
			return
		}

		var req dto.UpdateProfileRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.Error(apperror.NewBadRequestError(err.Error()))
			return
		}

		res, err := h.users.UpdateProfile(c.Request.Context(), userID, req)
		if err != nil {
			c.Error(err)
			return
		}
		c.JSON(http.StatusOK, res)
	}
}

func (h *userHandler) ChangePassword() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := utils.GetCurrentUserID(c)
		if err != nil {
			c.Error(err)
			return
		}

		var req dto.ChangePasswordRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.Error(apperror.NewBadRequestError(err.Error()))
			return
		}

		if err := h.users.ChangePassword(c.Request.Context(), userID, req); err != nil {
			c.Error(err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "password updated"})
	}
}
