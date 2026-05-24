package ws

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/pnhatthanh/vidio-guard-be/internal/apperror"
	"github.com/pnhatthanh/vidio-guard-be/internal/services"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type PipelineHandler struct {
	hub   *Hub
	token services.TokenService
}

func NewPipelineHandler(hub *Hub, token services.TokenService) *PipelineHandler {
	return &PipelineHandler{hub: hub, token: token}
}

func (h *PipelineHandler) HandlePipeline() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := h.authenticate(c)
		if err != nil {
			c.Error(err)
			return
		}

		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			c.Error(apperror.NewInternalServerError("websocket upgrade failed"))
			return
		}

		client := &Client{
			hub:    h.hub,
			conn:   conn,
			send:   make(chan []byte, 64),
			userID: userID.String(),
		}
		h.hub.registerClient(client)

		go client.writePump()
		go client.readPump()
	}
}

func (h *PipelineHandler) authenticate(c *gin.Context) (uuid.UUID, error) {
	token := strings.TrimSpace(c.Query("token"))
	if token == "" {
		auth := strings.TrimSpace(c.GetHeader("Authorization"))
		if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
			token = strings.TrimSpace(auth[7:])
		}
	}
	if token == "" {
		return uuid.Nil, apperror.NewUnauthorizedError("missing access token")
	}

	userID, jti, expiresAt, err := h.token.ValidateAccessToken(token)
	if err != nil {
		return uuid.Nil, apperror.NewUnauthorizedError("invalid token")
	}

	blacklisted, err := h.token.IsBlacklisted(c.Request.Context(), jti)
	if err != nil {
		return uuid.Nil, apperror.NewInternalServerError("failed to validate token")
	}
	if blacklisted {
		return uuid.Nil, apperror.NewUnauthorizedError("token revoked")
	}
	_ = expiresAt

	return userID, nil
}
