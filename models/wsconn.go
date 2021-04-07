package models

import (
	"context"
	"github.com/pkg/errors"

	"fmt"
	"github.com/gorilla/websocket"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	defaultTimeout = 1 * time.Minute
)

type wsConnWrapper interface {
	ReadMessage(ctx context.Context) (p []byte, err error)
	WriteMessage(ctx context.Context, data []byte) error
	Done() <-chan struct{}
	Close() error
}

const (
	connected   = iota
	closed      = iota
	retrying    = iota
	timedout    = iota
	interrupted = iota
)

type connectionInfo struct {
	Host     string `json:"host,omitempty"`
	HostID   string `json:"hostID,omitempty"`
	IsServer bool   `json:"isServer"`
}

type localMessage struct {
	msgType int
	data    []byte
}
type wsConnWrapperImpl struct {
	wsConn    *websocket.Conn
	connInfo  connectionInfo
	url       string
	headers   http.Header
	status    int
	retryStop chan struct{}

	done       chan struct{}
	maxTimeout time.Duration
	readMutex  sync.Mutex
	writeMutex sync.Mutex
	retryMutex sync.Mutex

	//sendQueue  chan *localMessage
}

/** returns a wsConn wrapper from a client connectingto a server */
func NewServerConn(ctx context.Context, url string, headers http.Header, maxTimeout time.Duration, connID string) (wsConnWrapper, error) {
	if maxTimeout == 0 {
		maxTimeout = defaultTimeout
	}
	timer := time.NewTimer(maxTimeout)
	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("context canceled [NewServerConn]")
		case <-timer.C:
			return nil, fmt.Errorf("Max timeout reached %s...", maxTimeout.String())
		default:
			c, _, err := websocket.DefaultDialer.Dial(url, headers)
			if err != nil {
				if !strings.Contains(err.Error(), "connection refused") {
					return nil, err
				}
			} else {
				return &wsConnWrapperImpl{wsConn: c,
					url: url, headers: headers,
					status:     connected,
					retryStop:  make(chan struct{}, 10),
					done:       make(chan struct{}, 10),
					maxTimeout: maxTimeout,
					connInfo: connectionInfo{
						Host:     GetOutboundIP(),
						HostID:   connID,
						IsServer: true,
					},
					//sendQueue: make(chan *localMessage, 1000),
				}, nil
			}
		}
	}
}

func NewClientConn(ctx context.Context, w http.ResponseWriter, r *http.Request, upgrader websocket.Upgrader, connID string) (wsConnWrapper, error) {
	wsConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, err
	}
	return &wsConnWrapperImpl{wsConn: wsConn,
		status: connected,
		done:   make(chan struct{}, 10),
		connInfo: connectionInfo{
			HostID:   connID,
			IsServer: false,
		},
		//sendQueue: make(chan *localMessage, 1000),
	}, nil
}

func (cc *wsConnWrapperImpl) retryConnection(ctx context.Context, loc string) error {
	if !cc.connInfo.IsServer {
		return fmt.Errorf("No retry for server")
	}
	cc.retryMutex.Lock()
	defer cc.retryMutex.Unlock()
	method := fmt.Sprintf("[retryConnection:%s]", loc)
	cc.status = retrying
	cc.wsConn = nil
	start := time.Now()
	for {
		select {
		case <-ctx.Done():
			cc.retryStop <- struct{}{}
			return fmt.Errorf("%s context canceled ", method)
		case <-cc.Done():
			cc.retryStop <- struct{}{}
			return fmt.Errorf("%s interrupted..", method)
		default:
			c, _, connErr := websocket.DefaultDialer.Dial(cc.url, cc.headers)
			if connErr != nil {
				// should wait until up
				log.Printf("%s Still error connecting to %s, connErr=%v\n", method, cc.url, connErr)
				if time.Since(start) > cc.maxTimeout {
					cc.status = timedout
					cc.retryStop <- struct{}{}
					return fmt.Errorf("%s -  failed to retry to connect to %s after %v seconds", method, cc.url, cc.maxTimeout.Seconds())
				}
				time.Sleep(1 * time.Second) // todo configurable
			} else {
				log.Println(method, "Got connection??? ")
				// otherwise go on
				cc.wsConn = c
				cc.status = connected
				cc.retryStop <- struct{}{}
				return nil
			}
		}
	}
}

func (cc *wsConnWrapperImpl) ReadMessage(ctx context.Context) (p []byte, err error) {
	const method = "ReadMessage"
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	for {
		select {
		case <-cc.Done():
			return nil, io.EOF
		case <-ctx.Done():
			return nil, fmt.Errorf("context canceled[ReadMessage]")
		default:
			if cc.wsConn != nil {
				//log.Printf("--------- ReadMessage  (connId=%s) isServer=%t",  cc.connInfo.HostID, cc.connInfo.IsServer )
				_, message, err := cc.wsConn.ReadMessage()
				// todo msgType that is not Text???
				if err != nil {
					// some one close this
					if strings.Contains(err.Error(), "closed network connection") {
						// got it.. just return
						cc.done <- struct{}{}
						return nil, io.EOF
					}

					log.Printf("ReadMessage got err = %v", err)
					if !cc.connInfo.IsServer || strings.Contains(err.Error(), "unexpected EOF") {
						return nil, err
					}
					if strings.Contains(err.Error(), "close") {
						if connErr := cc.retryConnection(ctx, method); connErr != nil {
							return nil, connErr
						}
						fmt.Printf("readMessageWithRetryConn == ?? ")
						if cc.wsConn != nil {
							_, message, err = cc.wsConn.ReadMessage()
							return message, err
						} else {
							// ???
							fmt.Printf("what?? ...")
						}
					}

				}
				return message, err
			}
			return nil, fmt.Errorf("reading from a close connection...")
		}
	}
}

func (cc *wsConnWrapperImpl) WriteMessage(ctx context.Context, data []byte) error {
	const method = "WriteMessage"
	if cc.status == retrying {
		<-cc.retryStop
	}
	if cc.status != connected {
		return fmt.Errorf("Connection has issue!!! bailed out, status=%d", cc.status)
	}
	if cc.wsConn == nil {
		return fmt.Errorf("Connection NIL - bailed out ..")
	}
	cc.writeMutex.Lock()
	defer cc.writeMutex.Unlock()
	log.Printf(">>>> %s - WriteMessage 1 %s", method, string(data))
	err := cc.wsConn.WriteMessage(websocket.TextMessage, data)
	if err == nil {
		return nil
	}
	if err == io.ErrClosedPipe || cc.connInfo.IsServer {
		return err
	}
	log.Println(method, " -- [2]", err)
	if strings.Contains(err.Error(), "close") {
		if connErr := cc.retryConnection(ctx, method); connErr != nil {
			return connErr
		}
		log.Printf(">>>> %s - WriteMessage 2", method)
		err = cc.wsConn.WriteMessage(websocket.TextMessage, data)
	}
	if err != nil {
		return errors.Wrapf(err, "connection error!")
	}
	return nil
}

func (cc *wsConnWrapperImpl) Close() error {
	// Cleanly close the connection by sending a close message and then
	// waiting (with timeout) for the server to close the connection.

	err := cc.tellTheOtherEndWeAreClosing()
	cc.status = closed
	return err
}
func (cc *wsConnWrapperImpl) tellTheOtherEndWeAreClosing() error {
	cc.writeMutex.Lock()
	defer cc.writeMutex.Unlock()
	if cc.wsConn != nil {
		cc.wsConn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		cc.wsConn.Close()
	}
	cc.wsConn = nil
	cc.done <- struct{}{}
	return nil
}
func (cc *wsConnWrapperImpl) Done() <-chan struct{} {
	return cc.done
}
