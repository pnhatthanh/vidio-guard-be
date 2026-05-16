package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pnhatthanh/vidio-guard-be/internal/apperror"
	"github.com/pnhatthanh/vidio-guard-be/internal/services"
)

type UploadHandler interface {
	UploadVideo() gin.HandlerFunc
}

type uploadHandler struct {
	videos services.VideoUploadService
}

func NewUploadHandler(videos services.VideoUploadService) UploadHandler {
	return &uploadHandler{videos: videos}
}

func (h *uploadHandler) UploadVideo() gin.HandlerFunc {
	return func(c *gin.Context) {
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
			reader,
			file.Size,
			file.Filename,
			file.Header.Get("Content-Type"),
		)
		if err != nil {
			c.Error(err)
			return
		}

		c.JSON(http.StatusOK, res)
	}
}
