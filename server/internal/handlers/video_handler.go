package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pnhatthanh/vidio-guard-be/internal/apperror"
	"github.com/pnhatthanh/vidio-guard-be/internal/services"
)

type VideoHandler interface {
	Upload() gin.HandlerFunc
	GetStatus() gin.HandlerFunc
}

type videoHandler struct {
	videos services.VideoService
}

func NewVideoHandler(videos services.VideoService) VideoHandler {
	return &videoHandler{videos: videos}
}

func (h *videoHandler) Upload() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := userIDFromContext(c)
		if err != nil {
			c.Error(err)
			return
		}

		file, err := c.FormFile("file")
		if err != nil {
			c.Error(apperror.NewBadRequestError("file is required"))
			return
		}

		reader, err := file.Open()
		if err != nil {
			c.Error(apperror.NewInternalServerError("could not read file"))
			return
		}
		defer reader.Close()

		res, err := h.videos.Upload(
			c.Request.Context(),
			userID,
			reader,
			file.Size,
			file.Filename,
			file.Header.Get("Content-Type"),
		)
		if err != nil {
			c.Error(err)
			return
		}

		c.JSON(http.StatusCreated, res)
	}
}

func (h *videoHandler) GetStatus() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := userIDFromContext(c)
		if err != nil {
			c.Error(err)
			return
		}

		videoID, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.Error(apperror.NewBadRequestError("invalid video id"))
			return
		}

		res, err := h.videos.GetStatus(c.Request.Context(), userID, videoID)
		if err != nil {
			c.Error(err)
			return
		}

		c.JSON(http.StatusOK, res)
	}
}

func userIDFromContext(c *gin.Context) (uuid.UUID, error) {
	raw, ok := c.Get("userID")
	if !ok {
		return uuid.Nil, apperror.NewUnauthorizedError("authentication required")
	}
	userID, ok := raw.(uuid.UUID)
	if !ok || userID == uuid.Nil {
		return uuid.Nil, apperror.NewUnauthorizedError("authentication required")
	}
	return userID, nil
}
