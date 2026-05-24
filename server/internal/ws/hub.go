package ws

import (
	"context"
	"log"
	"sync"

	"github.com/pnhatthanh/vidio-guard-be/internal/realtime"
)

type Hub struct {
	mu         sync.RWMutex
	clients    map[string]map[*Client]struct{}
	registerCh   chan *Client
	unregisterCh chan *Client
}

func NewHub() *Hub {
	return &Hub{
		clients:      make(map[string]map[*Client]struct{}),
		registerCh:   make(chan *Client),
		unregisterCh: make(chan *Client),
	}
}

func (h *Hub) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			h.closeAll()
			return
		case c := <-h.registerCh:
			h.addClient(c)
		case c := <-h.unregisterCh:
			h.removeClient(c)
		}
	}
}

func (h *Hub) BroadcastProgress(ev realtime.ProgressEvent) {
	payload, err := ev.Marshal()
	if err != nil {
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	set := h.clients[ev.UserID]
	for c := range set {
		c.enqueue(payload)
	}
}

func (h *Hub) addClient(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clients[c.userID] == nil {
		h.clients[c.userID] = make(map[*Client]struct{})
	}
	h.clients[c.userID][c] = struct{}{}
	log.Printf("[ws] client connected user=%s (total=%d)", c.userID, len(h.clients[c.userID]))
}

func (h *Hub) removeClient(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	set := h.clients[c.userID]
	if set == nil {
		return
	}
	if _, ok := set[c]; ok {
		delete(set, c)
		close(c.send)
	}
	if len(set) == 0 {
		delete(h.clients, c.userID)
	}
	log.Printf("[ws] client disconnected user=%s", c.userID)
}

func (h *Hub) closeAll() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for userID, set := range h.clients {
		for c := range set {
			close(c.send)
		}
		delete(h.clients, userID)
	}
}

func (h *Hub) registerClient(c *Client) {
	h.registerCh <- c
}

func (h *Hub) unregisterClient(c *Client) {
	h.unregisterCh <- c
}
