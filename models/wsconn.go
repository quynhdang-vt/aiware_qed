package models

import (
	"context"
	"errors"
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
	ReadMessage(ctx context.Context) (messageType int, p []byte, err error)
	WriteMessage(ctx context.Context, messageType int, data []byte) error
	Done() <-chan struct{}
	Close() error
}

const (
	connected = iota
	closed    = iota
	retrying  = iota
	timedout  = iota
)

type clientConn struct {
	wsConn           *websocket.Conn
	connInfo         connectionInfo
	url              string
	headers          http.Header
	status           int
	connectedChannel chan struct{}

	done             chan struct{}
	maxTimeout       time.Duration
	readMutex        sync.Mutex
	writeMutex       sync.Mutex
	retryMutex       sync.Mutex
}


func NewClientConn(ctx context.Context, url string, headers http.Header, maxTimeout time.Duration, connID string) (wsConnWrapper, error) {
	if maxTimeout == 0 {
		maxTimeout = defaultTimeout
	}
	timer := time.NewTimer(maxTimeout)
	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("context canceled [NewClientConn]")
		case <-timer.C:
			return nil, fmt.Errorf("Max timeout reached %s...", maxTimeout.String())
		default:
			c, _, err := websocket.DefaultDialer.Dial(url, headers)
			if err != nil {
				if !strings.Contains(err.Error(), "connection refused") {
					return nil, err
				}
			} else {
				return &clientConn{wsConn: c, url: url, headers: headers, status: connected, connectedChannel: make(chan struct{}, 10),
					done:       make(chan struct{}, 10),
					maxTimeout: maxTimeout,
					connInfo:   connectionInfo{
						Host:   GetOutboundIP(),
						HostID: connID,
					},
				}, nil
			}
		}
	}
}

type connectionInfo struct {
	Host string      `json:"host,omitempty"`
	HostID string    `json:"hostID,omitempty"`
}


func (cc *clientConn) retryConnection(ctx context.Context, loc string) error {
	cc.retryMutex.Lock()
	defer cc.retryMutex.Unlock()
	method := fmt.Sprintf("[retryConnection:%s]", loc)
	cc.status = retrying
	cc.wsConn = nil
	start := time.Now()
	for {
		select {
		case <-ctx.Done():
			cc.connectedChannel <- struct{}{}
			return fmt.Errorf("%s context canceled ", method)
		case <-cc.Done():
			cc.connectedChannel <- struct{}{}
			return fmt.Errorf("%s interrupted..", method)
		default:
			c, _, connErr := websocket.DefaultDialer.Dial(cc.url, cc.headers)
			if connErr != nil {
				// should wait until up
				log.Printf("%s Still error connecting to %s, connErr=%v\n", method, cc.url, connErr)
				if time.Since(start) > cc.maxTimeout {
					cc.status = timedout
					cc.connectedChannel <- struct{}{}
					return fmt.Errorf("%s -  failed to retry to connect to %s after %v seconds", method, cc.url, cc.maxTimeout.Seconds())
				}
				time.Sleep(1 * time.Second)
			} else {
				log.Println(method, "Got connection??? ")
				// otherwise go on
				cc.wsConn = c
				cc.status = connected
				cc.connectedChannel <- struct{}{}
				return nil
			}
		}
	}
}

func (cc *clientConn) ReadMessage(ctx context.Context) (messageType int, p []byte, err error) {
	const method = "ReadMessage"
	for {
		select {
		case <- cc.Done():
			return 0, nil, io.EOF
		case <-ctx.Done():
			return 0, nil, fmt.Errorf("context canceled[ReadMessage]")
		default:
			if cc.wsConn != nil {
				log.Println("--------- REMOVE ME REMOVE ME 100 times.. ReadMessage")
				msgType, message, err := cc.wsConn.ReadMessage()
				if err != nil {

					// some one close this
					if strings.Contains(err.Error(), "closed network connection") {
						// got it.. just return
						cc.stopReading()
						return 0, nil, io.EOF
					}

					log.Printf("ReadMessage got err = %v", err)

					if strings.Contains(err.Error(), "close") {

						if connErr := cc.retryConnection(ctx, method); connErr != nil {
							return 0, nil, connErr
						}
						fmt.Printf("readMessageWithRetryConn == ?? ")
						if cc.wsConn != nil {
							return cc.wsConn.ReadMessage()
						} else {
							// ???
							fmt.Printf("what?? ...")
						}
					}

				}
				return msgType, message, err
			}
			return 0, nil, errors.New("reading from a close connection...")
		}
	}
}
func (cc *clientConn) writeMessageSync(messageType int, data []byte) error {
	cc.writeMutex.Lock()
	defer cc.writeMutex.Unlock()
	if cc.wsConn != nil {
		return cc.wsConn.WriteMessage(messageType, data)
	}
	return io.ErrClosedPipe
}
func (cc *clientConn) WriteMessage(ctx context.Context, messageType int, data []byte) error {
	const method = "WriteMessage"
	if cc.status == retrying {
		<-cc.connectedChannel
	}
	if cc.status != connected {
		return fmt.Errorf("Connection has issue!!! bailed out, status=%d", cc.status)
	}

	err := cc.writeMessageSync(messageType, data)
	if err == nil {
		return nil
	}
	if err == io.ErrClosedPipe {
		return err
	}
	log.Println(method, " -- [1]", err)
	if strings.Contains(err.Error(), "close") {
		if connErr := cc.retryConnection(ctx, method); connErr != nil {
			return connErr
		}
		return cc.writeMessageSync(messageType, data)
	}

	return fmt.Errorf("connection error!")
}
func (cc *clientConn) Close() error {
	// Cleanly close the connection by sending a close message and then
	// waiting (with timeout) for the server to close the connection.
	err := cc.tellServerWeAreClosing()
	cc.done <- struct{}{}
	cc.status = closed
	return err
}
func (cc *clientConn) tellServerWeAreClosing() error {
	if cc.wsConn != nil {
		_ = cc.writeMessageSync(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		if cc.wsConn != nil {
			cc.wsConn.Close()
		}
		cc.wsConn = nil
	}
	return nil
}
func (cc *clientConn) Done() <-chan struct{} {
	return cc.done
}
func (cc *clientConn) stopReading() {
	cc.done <- struct{}{}
}