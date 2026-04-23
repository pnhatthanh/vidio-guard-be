package handlers

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pnhatthanh/vidio-guard-be/internal/pkg"
	"github.com/pnhatthanh/vidio-guard-be/internal/queue"
	"github.com/pnhatthanh/vidio-guard-be/internal/services"
)

type UploadHandler interface {
	UploadVideo() gin.HandlerFunc
	//GetStatus() gin.HandlerFunc
}

type uploadHandler struct {
	Uploader services.VideoUploadService
}

func NewUploadHandler(enqueuer queue.Enqueuer, store pkg.StoreProvider) UploadHandler {
	return &uploadHandler{
		Uploader: services.NewVideoUploadService(enqueuer, store),
	}
}

func (h *uploadHandler) UploadVideo() gin.HandlerFunc {
	return func(c *gin.Context) {
		file, err := c.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
			return
		}
		contentType := file.Header.Get("Content-Type")
		reader, err := file.Open()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not read file"})
			return
		}
		defer reader.Close()

		videoID, status, err := h.Uploader.Upload(c.Request.Context(), reader, file.Size, file.Filename, contentType)
		if err != nil {
			if errors.Is(err, services.ErrUploadStore) {
				log.Printf("[handler] failed uploading to object store: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "could not store file"})
				return
			}
			if errors.Is(err, services.ErrUploadEnqueue) {
				log.Printf("[handler] failed enqueuing task: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "could not queue job"})
				return
			}
			if errors.Is(err, services.ErrUploadInvalidInput) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid upload"})
				return
			}
			log.Printf("[handler] upload failed: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "upload failed"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"video_id": videoID,
			"status":   status,
		})
	}
}

// func (h *uploadHandler) GetStatus() gin.HandlerFunc {
// 	return func(c *gin.Context) {
// 		videoID := c.Param("video_id")
// 		if videoID == "" {
// 			c.JSON(http.StatusBadRequest, gin.H{"error": "video_id is required"})
// 			return
// 		}
// 		status, ok := h.StatusStore.GetStatus(videoID)
// 		if !ok {
// 			c.JSON(http.StatusNotFound, gin.H{"error": "video not found"})
// 			return
// 		}

// 		c.JSON(http.StatusOK, gin.H{
// 			"video_id": videoID,
// 			"status":   status,
// 		})
// 	}
// }
