package models

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"reflect"
	"time"
)

type message struct {
	MessageDataType string // so that we can properly deserialize based on the type
	Data            json.RawMessage
}

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

var typeRegistry = make (map[string]reflect.Type)
func registerType(o interface{}) {
	name, t := getTypeNameOfObject(o)
	log.Printf("registerType  REMOVE ME ... NAME=%s, t=%s", name, t.String())
	typeRegistry[name] = t
}
func getTypeNameOfObject(o interface{}) (string, reflect.Type) {
	t := reflect.TypeOf(o)
	return  t.String(), t
}

func init() {
	log.Printf("REMOVE ME moduleOnInit..")
	registerType(new (GetWorkRequest))
	registerType(new (GetWorkResponse))
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

func LocalInterfaceToRequestMsg(p interface{}) (RequestMsg, error) {
	b, _ := json.Marshal(p)
	name, _ := getTypeNameOfObject(p)
	msg := message{
		MessageDataType: name,
		Data:            b,
	}
	res, err := json.Marshal(msg)
	return RequestMsg{
		Timestamp: time.Now(),
		MsgType:   websocket.TextMessage,
		Data:      res,
	}, err
}

func ByteArrayToAType(msgType int, b []byte) (interface{}, error) {
	if msgType != websocket.TextMessage {
		return nil, fmt.Errorf("unrecognized %d", msgType)
	}
	var msg = &message{}
	err := json.Unmarshal(b, msg)
	if err != nil {
		return nil, err
	}
	log.Printf("-- REMOVE ME, MessageDataType =%s", msg.MessageDataType)
    p, err := makeInstance(msg.MessageDataType)
    if err != nil {
		log.Printf("-- REMOVE ME 0000, makeInstace(%s) failed? err=%v", msg.MessageDataType,err)
    	return p, err
	}
	log.Printf("-- REMOVE ME 1111, makeInstance(%s) type p = %s", msg.MessageDataType, reflect.TypeOf(p).String())

   n, t:= getTypeNameOfObject(p)

    log.Printf("----- REMOVE ME 2222, makeInstance... name=%s got type t=%v", n, t)
	err = json.Unmarshal(msg.Data, &p)
	log.Printf("--- REMOVE bytArrToCorrType failed? %v, type p =%s, p=%s", err, reflect.TypeOf(p).String(), ToString(p))
	return p, err
}

/*
func BytesToGetWorkRequest(msgType int, b []byte) (*GetWorkRequest, error) {
	p, err := byteArrToCorrectType(msgType, b)
	log.Printf(" Bytes to getWorkRequest -- err=%v, p=%s, type=%s", err, ToString(p), reflect.TypeOf(p).String())
	if err != nil {
		return nil, err
	}
	if res, ok := p.(*GetWorkRequest); ok {
		return res, nil
	}
	return nil, fmt.Errorf("Failed to assert to GetWorkRequest")
}
func BytesToGetWorkResponse(msgType int, b []byte) (*GetWorkResponse, error) {
	p, err := byteArrToCorrectType(msgType, b)
	if err != nil {
		return nil, err
	}
	if res, ok := p.(*GetWorkResponse); ok {
		return res, nil
	}
	return nil, fmt.Errorf("Failed to assert to GetWorkResponse")
}
*/
// GetCurrentTimeEpochMs returns current time UTC in milliseconds since epoch
func GetCurrentTimeEpochMs() int64 {
	return int64(time.Now().UTC().UnixNano() / 1e6)
}

type RequestMsg struct {
	Timestamp time.Time
	MsgType   int
	Data      []byte
}

type WSMessageHandlerFunc func(m RequestMsg) error
