package main

/*
go build
./server -addr localhost:8080
*/

import (
	"flag"
	"github.com/quynhdang-vt/aiware_qed/models"
	"log"
	"net/http"
)

func init() {
}

var addr = flag.String("addr", "localhost:8080", "http service address")

func main() {
	flag.Parse()
	log.SetFlags(0)

	wsHandler := NewWSHandle()
	wsHandler.AddHandler(models.ObjectTypeName(&models.GetWorkRequest{}), wsHandler.handleGetWorkRequest)
	http.HandleFunc("/getwork", wsHandler.ServeHTTP)

	log.Println("Server starting... at ", *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
}
