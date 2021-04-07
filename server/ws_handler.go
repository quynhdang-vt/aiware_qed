package main

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/quynhdang-vt/aiware_qed/models"
	"log"
	"net/http"
	"time"
)

// implement the
type WSHandler struct {
	upgrader      websocket.Upgrader
	ctx           context.Context
	wsConnectionManager *models.WebSocketConnectionHub
	myID          string
}

func NewWSHandle(ctx context.Context, wsConnectionManager *models.WebSocketConnectionHub) WSHandler {
	return WSHandler{
		upgrader:  websocket.Upgrader{},
	    wsConnectionManager: wsConnectionManager,

		ctx: ctx,
		myID : uuid.New().String(),
	}
}



// need context canceling!
func (h *WSHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.wsConnectionManager.AddClient(h.ctx, w, r, h.upgrader)
	// well need to wait for something??

}

func (h *WSHandler) handleGetWorkRequest(ctx context.Context, m *models.MessageInfo) error {
	/// --- do something --- check against the list or just plain store in the map to say that an engine instance wants work
	// for now, assumes that we'll just turn around and send it on
	o := m.Object
	if p, b := o.(*models.GetWorkRequest); b {
		// of course to do something about this:
		workResponse := &models.GetWorkResponse{
			Name:         p.Name,
			ID:           p.ID,
			ConnID:       p.ConnID,
			TimestampUTC: models.GetCurrentTimeEpochMs(),
			WorkItem:     fmt.Sprintf("Server %s - Work ITEM hand to you at %s", h.myID, time.Now().Format(time.RFC3339)),
		}

		log.Printf("WSHandler.handleGetWorkRequest - sending %s", models.ToString(workResponse))
		return h.wsConnectionManager.PublishToOneDestination(ctx, workResponse, m.ServerID)

	} else {
		log.Printf("test test test ... ")
		return fmt.Errorf("not a getworkrequest")
	}
}
