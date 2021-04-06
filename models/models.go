package models

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	controllerClient "github.com/veritone/realtime/modules/controller/client"
	"log"
	"reflect"
	"time"
)

type GetWorkResponse struct {
	Name         string `json:"name,omitempty"`
	ConnID       string `json:"connID,omitempty"`
	ID           string `json:"id,omitempty`
	TimestampUTC int64  `json:"timestampUTC,omitempty"` // milliseconds since epoch
	WorkItem     string `json:"workItem,omitempty"`
}

type GetWorkRequest struct {
	Name         string `json:"name,omitempty"`
	ConnID       string `json:"connID,omitempty"`
	ID           string `json:"id,omitempty`
	TimestampUTC int64  `json:"timestampUTC,omitempty"` // milliseconds since epoch
	TTL          int64  `json:"ttl,omitempty"`
}

var typeRegistry = make(map[string]reflect.Type)

func registerType(o interface{}) {
	name, t := getTypeNameOfObject(o)
	log.Printf("registerType  REMOVE ME ... NAME=%s, t=%s", name, t.String())
	typeRegistry[name] = t
}
func getTypeNameOfObject(o interface{}) (string, reflect.Type) {
	t := reflect.TypeOf(o)
	return t.String(), t
}

func ObjectTypeName(o interface{}) string {
	return reflect.TypeOf(o).String()
}

func init() {
	registerTypes()
}
func registerTypes() {
	log.Printf("REMOVE ME moduleOnInit..")
	registerType(new(GetWorkRequest))
	registerType(new(GetWorkResponse))
	registerType(new(controllerClient.EngineInstanceWorkRequest))
	registerType(new(controllerClient.EngineInstanceWorkRequestResponse))
}

//https://stackoverflow.com/questions/45679408/unmarshal-json-to-reflected-struct
func makeInstance(name string) (interface{}, error) {
	if myType, found := typeRegistry[name]; found {
		p := reflect.New(myType.Elem()).Interface()
		log.Printf("--- REMOVE ME makeInstance (%s), get %s", name, reflect.TypeOf(p).String())
		return p, nil
	} else {
		return nil, fmt.Errorf("%s is not registered", name)
	}
}

func ToString(c interface{}) string {
	if c == nil {
		return ""
	}
	s, _ := json.MarshalIndent(c, "", "\t")
	return string(s)
}

func SerializeToBytesForTransport(p interface{}) ([]byte, error) {
	b, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}
	name, _ := getTypeNameOfObject(p)
	log.Printf("SerializeToBytesForTransport : %s", name)
	res := RequestMsg{
		Timestamp: time.Now(),
		MsgType:   name,
		Data:      b,
	}
	return json.Marshal(res)
}

/*
ByteArrayToAType deserializes the byte araray into RequestMsg then from the Data unmarshalled to the real object

*/
func ByteArrayToAType(msgType int, b []byte) (interface{}, error) {
	if msgType != websocket.TextMessage {
		return nil, fmt.Errorf("unrecognized %d", msgType)
	}
	msg := RequestMsg{}
	err := json.Unmarshal(b, &msg)
	if err != nil {
		return nil, err
	}
	log.Printf("------ByteArrayToAType, %s", msg.MsgType)
	p, err := makeInstance(msg.MsgType)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(msg.Data, &p)
	return p, err
}

// GetCurrentTimeEpochMs returns current time UTC in milliseconds since epoch
func GetCurrentTimeEpochMs() int64 {
	return int64(time.Now().UTC().UnixNano() / 1e6)
}

type RequestMsg struct {
	Timestamp time.Time
	MsgType   string
	Data      []byte
}

type WSMessageHandlerFunc func(ctx context.Context, m interface{}) error
