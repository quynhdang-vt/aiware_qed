package main

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/quynhdang-vt/aiware_qed/models"
	"log"
	"net/http"
	"sync"
	"time"
)

// implement the
type WSHandler struct {
	upgrader websocket.Upgrader
	ctx      context.Context
	wsHub    *models.WebSocketConnectionHub
	myID     string
}

func NewWSHandle(ctx context.Context) WSHandler {
	wsHub := models.NewWebSocketHub()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		wsHub.Run(ctx)
	}()

	return WSHandler{
		upgrader: websocket.Upgrader{},
		wsHub:    wsHub,

		ctx:  ctx,
		myID: uuid.New().String(),
	}
}

func (h *WSHandler) AddHandler(objectType string, handler models.WSMessageHandlerFunc) {
	h.wsHub.AddHandler(objectType, handler)

}

// need context canceling!
func (h *WSHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.wsHub.AddClient(h.ctx, w, r, h.upgrader)
}

func (h *WSHandler) handleGetWorkRequest(ctx context.Context, m *models.MessageInfo) error {
	/// --- do something --- check against the list or just plain store in the map to say that an engine instance wants work
	// for now, assumes that we'll just turn around and send it on
	if m == nil || m.Object == nil {
		return fmt.Errorf("invalid parameter")
	}

	if p, b := m.Object.(*models.GetWorkRequest); b {
		// of course to do something about this:
		workResponse := &models.GetWorkResponse{
			Name:         p.Name,
			ID:           p.ID,
			ConnID:       p.ConnID,
			TimestampUTC: models.GetCurrentTimeEpochMs(),
			WorkItem:     fmt.Sprintf("Server %s - Work ITEM hand to you at %s", h.myID, time.Now().Format(time.RFC3339)),
		}

		log.Printf("WSHandler.handleGetWorkRequest - sending %s", models.ToString(workResponse))
		return h.wsHub.PublishToOneDestination(ctx, workResponse, m.ServerID)

	} else {
		log.Printf("test test test ... ")
		return fmt.Errorf("not a getworkrequest")
	}
}
