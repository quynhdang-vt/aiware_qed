package models

import (
	"context"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"time"
)

/**
WorkManager connects to an agent proxy WS endpoint -- if available,
and push getWorkRequest and wait for getWorkResponse (as soon as agent has it)
*/
type WorkManager struct {
	myConn wsConnWrapper
	url    string
	myid   string

	handlers map[string]WSMessageHandlerFunc
}

//(ctx context.Context, url string, headers http.Header, maxTimeout time.Duration) (wsConnWrapper, error)
func NewWorkManager(ctx context.Context, url string, headers http.Header, maxTimeout time.Duration, connID string) (*WorkManager, error) {
	conn, err := NewClientConn(ctx, url, headers, maxTimeout, connID)
	if err != nil {
		return nil, err
	}
	return &WorkManager{
		myConn:   conn,
		url:      url,
		handlers: make(map[string]WSMessageHandlerFunc),
	}, nil
}
func (w *WorkManager) GettURL() string {
	return w.url
}
func (w *WorkManager) Close() {
	w.myConn.Close()
}

func (w *WorkManager) AddHandler(typeName string, handler WSMessageHandlerFunc) {
	w.handlers[typeName] = handler
}
func (w *WorkManager) Consume(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	msg := make(chan interface{}, 500)
	errChan := make(chan error, 500)

	go func() {
		defer close(msg)
		defer w.myConn.Close()
		for {
			select {
			case <-ctx.Done():
				return
			case <-w.myConn.Done():
				return
			default:
				mtype, data, err := w.myConn.ReadMessage(ctx)
				if err == nil {
					var p interface{}
					p, err = ByteArrayToAType(mtype, data)
					msg <- p
				} else {
					errChan <- err
				}
			}
		}
	}()

	for m := range msg {
		// look up handler
		typeName, _ := getTypeNameOfObject(m)
		if h, b := w.handlers[typeName]; b {
			h(ctx, m)
		} else {
			// just log
			log.Printf("Got %v", m)
		}
	}

}

func (w *WorkManager) Publish(ctx context.Context, m []byte) error {
	return w.myConn.WriteMessage(ctx, websocket.TextMessage, m)
}
