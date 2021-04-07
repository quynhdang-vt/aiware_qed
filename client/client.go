package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/google/uuid"
	"github.com/quynhdang-vt/aiware_qed/models"
	"log"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"time"
)

var addr1 = flag.String("addr1", "localhost:8080", "http service address")
var addr2 = flag.String("addr2", "localhost:8090", "http service address")


func main() {
	flag.Parse()
	log.SetFlags(0)

	interrupt := make(chan os.Signal, 100)
	signal.Notify(interrupt, os.Interrupt)

	wm := models.NewWebSocketHub()
	u1 := url.URL{Scheme: "ws", Host: *addr1, Path:  models.WSEndpoint}
	u2 := url.URL{Scheme: "ws", Host: *addr2, Path: models.WSEndpoint}
	serverID1 := uuid.New().String()
	serverID2 := uuid.New().String()


	ctx, cancel := context.WithCancel(context.Background())

	wm.AddServer(ctx, u1.String(), nil, 0, serverID1)
	wm.AddServer(ctx, u2.String(), nil, 0, serverID2)

	/**
	a handler is given context and an interface pointer to
	 */
	getWorkResponseHandler := func(ctx context.Context, aMsg *models.MessageInfo) error {
		if aMsg.Object == nil {
			return fmt.Errorf("Object is NIL - no good")
		}
		if p, b := aMsg.Object.(*models.GetWorkResponse); b {
			// do something...
			log.Printf("getWorkResponseHandler got %s", models.ToString(p))
		} else {
			log.Printf("getWorkResponseHandler got err")
		}
		return nil
	}
	wm.AddHandler(models.ObjectTypeName(&models.GetWorkResponse{}), getWorkResponseHandler )

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		wm.Run(ctx)
	}()

	// let's say we want to send a getwork
	connID := uuid.New().String()
	for i := 0; i < 5; i++ {

		timestring := time.Now().Format(time.RFC3339)
		obj := &models.GetWorkRequest{
			Name:         fmt.Sprintf("NAME-%d-@%s", i, timestring),
			ID:           fmt.Sprintf("ID-%d-@%s", i, timestring),
			ConnID:       connID,
			TimestampUTC: models.GetCurrentTimeEpochMs(),
			TTL:          3600,
		}

		wm.Broadcast(ctx, obj)
		time.Sleep(2 * time.Second)

	}
	// let's go down to 1
	wm.Close(serverID1)

	timestring := time.Now().Format(time.RFC3339)
	obj := &models.GetWorkRequest{
		Name:         fmt.Sprintf("NAME-TO SERVER 2 ONLY-@%s",  timestring),
		ID:           fmt.Sprintf("ID-TO SERVER 2-@%s", timestring),
		ConnID:       connID,
		TimestampUTC: models.GetCurrentTimeEpochMs(),
		TTL:          3600,
	}


	wm.PublishToOneDestination(ctx, obj, serverID2)

	time.Sleep(5*time.Second)

	// now canceling and close up
	// ----- TEST1
	// how to figure out that we need to flush everything
	// cancel too soon and we'll die...
	cancel()
	wm.RemoveAll()
	log.Println("----- in MAIN Waiting for every one to close up shop..")

	wg.Wait()
	log.Println("The END")
}
