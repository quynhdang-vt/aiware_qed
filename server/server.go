package main

/*
go build
./server -addr localhost:8080
*/

import (
	"context"
	"flag"
	"github.com/quynhdang-vt/aiware_qed/models"
	"log"
	"net/http"
	"sync"
)

func init() {
}

var addr = flag.String("addr", "localhost:8080", "http service address")

func main() {
	flag.Parse()
	log.SetFlags(0)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wsHub := models.NewWebSocketHub()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		wsHub.Run(ctx)
	}()

	wsHandler := NewWSHandle(ctx, wsHub)
	wsHub.AddHandler(models.ObjectTypeName(&models.GetWorkRequest{}), wsHandler.handleGetWorkRequest)

	http.HandleFunc(models.WSEndpoint, wsHandler.ServeHTTP)

	log.Println("Server starting... at ", *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
}
