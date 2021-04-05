package models

import (
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
)

func TestGetWorkRequestToMessageAndBack(t *testing.T) {
	//init()
	p := GetWorkRequest{
		Name:         "name1",
		ID:           "id1",
		TTL:          1000,
		TimestampUTC: GetCurrentTimeEpochMs(),
	}

	reqMsg, err := LocalInterfaceToRequestMsg(&p)
	assert.Contains(t, string(reqMsg.Data), reflect.TypeOf(p).String())
	t.Logf("msgType=%d, %s, err=%v", reqMsg.MsgType, string(reqMsg.Data), err)
	pa2, err := ByteArrayToAType(reqMsg.MsgType, reqMsg.Data)
	assert.Nil(t, err)
	assert.NotNil(t, pa2)
	p2, ok := pa2.(*GetWorkRequest)
	assert.True(t, ok)
	assert.NotNil(t, p2)
	t.Logf("p2 = %s, err =%v", ToString(p2), err)
	assert.Equal(t, p.Name, p2.Name)
	assert.Equal(t, p.ID, p2.ID)
	assert.Equal(t, p.TimestampUTC, p2.TimestampUTC)
	assert.Equal(t, p.TTL, p2.TTL)

	p3, ok :=  pa2.(*GetWorkResponse)
	assert.False(t, ok)
	assert.Nil(t, p3)
}

func TestGetWorkResponseToMessageAndBack(t *testing.T) {
	p := GetWorkResponse{
		Name:         "name1",
		ID:           "id1",
		WorkItem:     "workItem1",
		TimestampUTC: GetCurrentTimeEpochMs(),
	}

	reqMsg, err := LocalInterfaceToRequestMsg(&p)
	assert.Contains(t, string(reqMsg.Data), reflect.TypeOf(p).String())
	t.Logf("msgType=%d, %s, err=%v", reqMsg.MsgType, string(reqMsg.Data), err)
	pa2, err := ByteArrayToAType(reqMsg.MsgType, reqMsg.Data)
	assert.Nil(t, err)
	assert.NotNil(t, pa2)
	p2, ok := pa2.(*GetWorkResponse)
	assert.True(t, ok)
	assert.NotNil(t, p2)
	t.Logf("p2 = %s, err =%v", ToString(p2), err)
	assert.Equal(t, p.Name, p2.Name)
	assert.Equal(t, p.ID, p2.ID)
	assert.Equal(t, p.TimestampUTC, p2.TimestampUTC)
	assert.Equal(t, p.WorkItem, p2.WorkItem)

	p3, ok :=  pa2.(*GetWorkRequest)
	assert.False(t, ok)
	assert.Nil(t, p3)
}
