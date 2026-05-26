package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pnhatthanh/vidio-guard-be/internal/apperror"
	"github.com/pnhatthanh/vidio-guard-be/internal/dto"
	"github.com/pnhatthanh/vidio-guard-be/internal/services"
	"github.com/pnhatthanh/vidio-guard-be/internal/utils"
)

const maxProfileMultipartBytes = 6 << 20 // 6 MB (avatar limit 5 MB + overhead)

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

		if err := c.Request.ParseMultipartForm(maxProfileMultipartBytes); err != nil {
			c.Error(apperror.NewBadRequestError("invalid multipart form"))
			return
		}

		input := dto.UpdateProfileInput{
			FullName:     c.PostForm("full_name"),
			RemoveAvatar: strings.EqualFold(strings.TrimSpace(c.PostForm("remove_avatar")), "true"),
		}

		file, err := c.FormFile("avatar")
		if err == nil && file != nil {
			reader, err := file.Open()
			if err != nil {
				c.Error(apperror.NewInternalServerError("could not read avatar"))
				return
			}
			defer reader.Close()

			input.HasAvatar = true
			input.AvatarReader = reader
			input.AvatarSize = file.Size
			input.AvatarFilename = file.Filename
			input.AvatarContentType = file.Header.Get("Content-Type")
		}

		res, err := h.users.UpdateProfile(c.Request.Context(), userID, input)
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
