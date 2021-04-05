package main

/*
go build
./server -addr localhost:8080 
*/

import (
	"flag"
	"fmt"
	"github.com/google/uuid"
	"github.com/quynhdang-vt/aiware_qed/models"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

func init() {
}
var addr = flag.String("addr", "localhost:8080", "http service address")
var myID = uuid.New().String()
var upgrader = websocket.Upgrader{} // use default options

func getwork(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	log.Println("----------------------- STARTING a WS CONN.........")
	defer c.Close()
	// expectation:  client push getwork request and we send on getworkResponse
	for {
		mt, message, err := c.ReadMessage()
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
		 p, b := wr.(*models.GetWorkRequest)
		 if !b {
			log.Println(" Not getting type type..")
			continue
		}
		/// --- do something --- check against the list or just plain store in the map to say that an engine instance wants work
		// for now, assumes that we'll just turn around and send it on

		workResponse := &models.GetWorkResponse{
			Name:         p.Name,
			ID:           p.ID,
			ConnID:       p.ConnID,
			TimestampUTC: models.GetCurrentTimeEpochMs(),
			WorkItem:     fmt.Sprintf("Server %s - Work ITEM hand to you at "+time.Now().Format(time.RFC3339), myID),
		}

		reqMsg, err := models.LocalInterfaceToRequestMsg(workResponse)
		if err != nil {
			// skip
			log.Println("Fail to marshal work response, err=%v", err)
			continue
		}
		log.Println("PUSHING Work Response: ", models.ToString(workResponse))
		err = c.WriteMessage(reqMsg.MsgType, reqMsg.Data)
		if err != nil {
			log.Println("write:", err)
			break
		}
	}
}


func main() {
	flag.Parse()
	log.SetFlags(0)
	http.HandleFunc("/getwork", getwork)

	log.Println("Server starting... at ", *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
}
