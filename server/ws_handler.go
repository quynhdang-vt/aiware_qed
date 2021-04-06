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
	upgrader websocket.Upgrader
	myID     string
	handlers map[string]models.WSMessageHandlerFunc
	wsConn   *websocket.Conn
}

func NewWSHandle() WSHandler {
	return WSHandler{
		upgrader: websocket.Upgrader{},
		myID:     uuid.New().String(),
		handlers: make(map[string]models.WSMessageHandlerFunc),
	}
}
func (h *WSHandler) AddHandler(typeName string, handler models.WSMessageHandlerFunc) {
	h.handlers[typeName] = handler
}
func (h *WSHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var err error
	h.wsConn, err = h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	log.Println("----------------------- STARTING a WS CONN.........")
	defer h.wsConn.Close()
	// expectation:  client push getwork request and we send on getworkResponse
	messageQueue := make(chan interface{}, 100)

	go func () {
		for m := range messageQueue {
			// look up handler
			typeName := models.ObjectTypeName(m)
			if handler, b := h.handlers[typeName]; b {
				handler(context.Background(), m)
			} else {
				// just log
				log.Printf("Got %v", m)
			}
		}
	} ()
	for {
		mt, message, err := h.wsConn.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			break
		}

		wr, err := models.ByteArrayToAType(mt, message)
		if err != nil {
			// skip
			log.Println("RECEIVING NOT a work request, err=", err)
			continue
		}
		log.Printf("Got %s", models.ToString(wr))
		messageQueue <- wr
	}

}

func (h *WSHandler) handleGetWorkRequest(ctx context.Context, o interface{}) error {
	/// --- do something --- check against the list or just plain store in the map to say that an engine instance wants work
	// for now, assumes that we'll just turn around and send it on
	if p, b := o.(*models.GetWorkRequest); b {
		workResponse := &models.GetWorkResponse{
			Name:         p.Name,
			ID:           p.ID,
			ConnID:       p.ConnID,
			TimestampUTC: models.GetCurrentTimeEpochMs(),
			WorkItem:     fmt.Sprintf("Server %s - Work ITEM hand to you at "+time.Now().Format(time.RFC3339), h.myID),
		}

		bArr, err := models.SerializeToBytesForTransport(workResponse)
		if err != nil {
			// skip
			log.Println("Fail to marshal work response, err=%v", err)
			return err
		}
		log.Println("PUSHING Work Response: ", models.ToString(workResponse))
		if h.wsConn != nil {
			err = h.wsConn.WriteMessage(websocket.TextMessage, bArr)
			if err != nil {
				log.Println("write:", err)
				return err
			}
			return nil
		} else {
			return fmt.Errorf("NO WSCONN to return??")
		}
	} else {
		log.Printf("test test test ... ")
		return fmt.Errorf("not a getworkrequest")
	}
}
