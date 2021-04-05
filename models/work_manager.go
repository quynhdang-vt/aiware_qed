package models

import (
	"context"
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
}

//(ctx context.Context, url string, headers http.Header, maxTimeout time.Duration) (wsConnWrapper, error)
func NewWorkManager(ctx context.Context, url string, headers http.Header, maxTimeout time.Duration, connID string) (*WorkManager, error) {
	conn, err := NewClientConn(ctx, url, headers, maxTimeout, connID)
	if err != nil {
		return nil, err
	}
	return &WorkManager{
		myConn: conn,
		url:    url,
	}, nil
}
func (w *WorkManager) GettURL() string {
	return w.url;
}
func (w *WorkManager) Close () {
	w.myConn.Close()
}
func (w *WorkManager) Consume(ctx context.Context, handler WSMessageHandlerFunc) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	msg := make(chan RequestMsg, 500)
	errChan := make (chan error, 500)

	go func() {
		defer 				close(msg)
		defer 	w.myConn.Close()
		for {
			select {
			case <-ctx.Done():
				log.Println("---- REMOVE ME 1")
				return
			case <-w.myConn.Done():
				log.Println("---- REMOVE ME 2")
				return
			default:
				mtype, data, err := w.myConn.ReadMessage(ctx)
				if err == nil {
					m := RequestMsg{Timestamp: time.Now(), MsgType: mtype, Data: data}
					log.Println("WorkManager.Consume  -- ", string(m.Data))
					msg <- m
				} else {
					errChan <- err
				}
			}
		}
	}()

	for m := range msg {
		if handler != nil {
			log.Println("HERE I AM")
			handler(m)
		} else {
			// just log
			log.Printf("Got msg @ %s msgType=%d, data=%s", m.Timestamp.Format(time.RFC3339), m.MsgType, string(m.Data))
		}
	}

}

func (w *WorkManager) Publish(ctx context.Context, m RequestMsg) error {
	log.Println("WorkManager.Publish -- ", ToString(m))
	return w.myConn.WriteMessage(ctx, m.MsgType, m.Data)
}
