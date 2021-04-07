package models

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"log"
	"net/http"
	"time"
)

/**
WebSocketConnectionHub manages websocket connections to different servers.
y
*/

type ConnectionInfo struct {
	myConn   wsConnWrapper
	url      string
	serverID string
	done     chan struct{}
}

type WebSocketConnectionHub struct {
	serverInfo map[string]*ConnectionInfo
	myid       string
	isClosed   bool

	handlers map[string]WSMessageHandlerFunc

	inQueue chan *MessageInfo
	errChan chan *MessageInfo
}

//(ctx context.Context, url string, headers http.Header, maxTimeout time.Duration) (wsConnWrapper, error)
func NewWebSocketHub() *WebSocketConnectionHub {
	return &WebSocketConnectionHub{
		myid:       uuid.New().String(),
		handlers:   make(map[string]WSMessageHandlerFunc),
		inQueue:    make(chan *MessageInfo, 1000),
		errChan:    make(chan *MessageInfo, 1000),
		serverInfo: make(map[string]*ConnectionInfo),
	}
}

/**
AddServer starts a websocket connection to a server
*/
func (w *WebSocketConnectionHub) AddServer(ctx context.Context, url string, headers http.Header, maxTimeout time.Duration, connID string) error {
	conn, err := NewServerConn(ctx, url, headers, maxTimeout, connID)
	if err != nil {
		return err
	}
	w.serverInfo[connID] = &ConnectionInfo{myConn: conn, url: url, serverID: connID, done: make(chan struct{}, 10)}
	// as soon as a connection is added, we should start consuming from it
	go w.connectionLoop(ctx, connID)
	return nil
}

/**
AddClient starts a websocket connection from a client to this server as connected  via HTTP handler
*/
func (w *WebSocketConnectionHub) AddClient(ctx context.Context, httpWriter http.ResponseWriter, httpReq *http.Request, upgrader websocket.Upgrader) (retErr error) {
	const method = "websocketConnectionHub.AddClient"
	log.Printf("%s ENTER", method)
	defer func() {
		log.Printf("%s rETUrning %v", method, retErr)
	}()
	connID := httpReq.RemoteAddr
	conn, err := NewClientConn(ctx, httpWriter, httpReq, upgrader, connID)
	if err != nil {
		return err
	}
	w.serverInfo[connID] = &ConnectionInfo{
		myConn:   conn,
		serverID: connID,
		done:     make(chan struct{}, 10),
	}
	go w.connectionLoop(ctx, connID)
	return nil
}

func (w *WebSocketConnectionHub) Close(serverID string) error {
	if server, b := w.serverInfo[serverID]; b && server != nil {
		if server.myConn != nil {
			server.myConn.Close()
			server.myConn = nil
			server.done <- struct{}{}
			w.serverInfo[serverID] = server
		}
		return nil
	}
	return fmt.Errorf("%s not exists", serverID)
}

func (w *WebSocketConnectionHub) Remove(serverID string) {
	if w.Close(serverID) == nil {
		delete(w.serverInfo, serverID)
	}
}
func (w *WebSocketConnectionHub) CloseAll() {
	for k, server := range w.serverInfo {
		if server.myConn != nil {
			server.myConn.Close()
			server.myConn = nil
			server.done <- struct{}{}
			w.serverInfo[k] = server
		}
	}
}
func (w *WebSocketConnectionHub) RemoveAll() {
	for k, _ := range w.serverInfo {
		w.Remove(k)
	}
	w.isClosed = true
	w.closeInQueue()
	w.closeErrChan()
}
func (w *WebSocketConnectionHub) AddHandler(typeName string, handler WSMessageHandlerFunc) {
	w.handlers[typeName] = handler
}

func (w *WebSocketConnectionHub) closeInQueue() {
	close(w.inQueue)
}
func (w *WebSocketConnectionHub) closeErrChan() {
	close(w.errChan)
}

/* do we want to get individual ?

 */
func (w *WebSocketConnectionHub) connectionLoop(ctx context.Context, serverID string) error {
	const method = "connectionLoop"
	var theServer *ConnectionInfo
	var found bool
	if theServer, found = w.serverInfo[serverID]; !found || theServer == nil {
		return fmt.Errorf("%s NOT FOUND", serverID)
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	// do we want to close??
	defer w.Close(serverID)

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-theServer.done:
			return nil
		default:
			data, err := theServer.myConn.ReadMessage(ctx)
			if err == nil {
				var p interface{}
				p, err = ByteArrayToAType(data)
				log.Printf("%s - ReadMessage from theServer connID=%s p=%s, err=%v", method, theServer.serverID, ToString(p), err)
				if err == nil && p != nil && !w.isClosed {
					w.inQueue <- &MessageInfo{
						ServerID: serverID,
						Object:   p,
					}
				}
			}
			if err != nil {
				if !w.isClosed {
					w.errChan <- &MessageInfo{
						ServerID: serverID,
						Object:   err,
					}
				}
				return err
			}
		}
	}
}

/**
Run starts dispatching work from the inQueue to handlers
*/
func (w *WebSocketConnectionHub) Run(ctx context.Context) {
	const method = "WebSocketConnectionHub.Run"
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go w.errorChecking()
	for {
		select {
		case <-ctx.Done():
			return
		case m := <-w.inQueue:
			// look up handler
			typeName := ObjectTypeName(m.Object)
			if h, b := w.handlers[typeName]; b {
				if err := h(ctx, m); err != nil && !w.isClosed {
					w.errChan <- &MessageInfo{m.ServerID, errors.Wrapf(err, "Failed to handle message from server %s", m.ServerID)}
				}
			} else {
				// just log
				log.Printf("Got %v", m)
			}
		}
	}
}
func (w *WebSocketConnectionHub) errorChecking() {
	// consume from error channel
	for e := range w.errChan {
		// Object is an error so for now we'll just log away
		log.Printf("WebSocketConnectionHub got err = %v", e)
	}
}

func (w *WebSocketConnectionHub) PublishToOneDestination(ctx context.Context, o interface{}, serverID string) error {
	var method = fmt.Sprintf("PublishToOneDestination:%s", serverID)
	defer log.Printf("%s EXIT", method)
	bArr, err := SerializeToBytesForTransport(o)
	if err != nil {
		return err
	}
	if server, b := w.serverInfo[serverID]; b && server != nil {

		if server.myConn != nil {
			log.Printf("%s GO ON...", method)
			return server.myConn.WriteMessage(ctx, bArr)
		} else {
			return fmt.Errorf("server %s connection NIL, no publishing", serverID)
		}
	}
	return fmt.Errorf("server %s not found for publishing", serverID)
}

func (w *WebSocketConnectionHub) Broadcast(ctx context.Context, o interface{}) error {
	bArr, err := SerializeToBytesForTransport(o)
	if err != nil {
		return err
	}
	for serverID, server := range w.serverInfo {
		log.Printf("Broadcast to %s", serverID)
		if server.myConn != nil {
			if err := server.myConn.WriteMessage(ctx, bArr); err != nil && !w.isClosed {
				w.errChan <- &MessageInfo{ServerID: server.serverID, Object: err}
			}
		} else {
			if !w.isClosed {
				w.errChan <- &MessageInfo{ServerID: server.serverID, Object: fmt.Errorf("server %s connection NIL, no publishing", server.serverID)}
			}
		}

	}
	return nil
}
