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

func getWorkManager(url string) (*models.WorkManager, error) {
	log.Printf("connecting to %s", url)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	connID := uuid.New().String()

	workManager, err := models.NewWorkManager(ctx, url, nil, 0, connID)

	return workManager, err
}
func main() {
	flag.Parse()
	log.SetFlags(0)

	interrupt := make(chan os.Signal, 100)
	signal.Notify(interrupt, os.Interrupt)

	u1 := url.URL{Scheme: "ws", Host: *addr1, Path: "/getwork"}
	wm1, err1 := getWorkManager(u1.String())
	u2 := url.URL{Scheme: "ws", Host: *addr2, Path: "/getwork"}
    wm2, err2 := getWorkManager(u2.String())

    if err1 != nil && err2 != nil {
    	log.Fatalf("NO server...")
	}

	ctx, cancel := context.WithCancel(context.Background())

	var mapMutex sync.RWMutex
	workMap := make(map[string]*models.GetWorkResponse)
	getWorkResponseHandler := func(m models.RequestMsg) error {
		// make sure that it is something we can do
		wr, err := models.BytesToGetWorkResponse(m.MsgType, m.Data)
		if err == nil {
			mapMutex.Lock()
			workMap[wr.ID] = wr
			mapMutex.Unlock()
			log.Printf("getWorkResponseHandler got %s", models.ToString(wr))
		} else {
			log.Printf("getWorkResponseHandler got err? %v", err)
		}
		return nil
	}
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer log.Printf("WM1 - Exiting receiving work responses, %s\n", wm1.GettURL())
		wm1.Consume(ctx, getWorkResponseHandler)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer log.Printf("WM2 - Exiting receiving work responses, %s\n", wm2.GettURL())
		wm2.Consume(ctx, getWorkResponseHandler)
	}()
	getWork := make(chan models.RequestMsg, 100)
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer log.Println("Exiting pushing work requests")
		for {
			select {
			case <-ctx.Done():
				return
			case m := <-getWork:
				log.Println("Push a getwork request")
				err := wm1.Publish(ctx, m)
				if err != nil {
					log.Printf("WM1 write to %s, got err=%v", wm1.GettURL(), err)
				}
				err  = wm2.Publish(ctx, m)
				if err != nil {
					log.Printf("WM2 write to %s, got err=%v", wm2.GettURL(), err)
				}
			}
		}
	}()

	// let's say we want to send a getwork
	connID := uuid.New().String()
	for i:=0; i<5; i++ {

		timestring := time.Now().Format(time.RFC3339)
		if msg1, err := models.LocalInterfaceToRequestMsg(models.GetWorkRequest{
			Name:         fmt.Sprintf("NAME-%d-@%s", i, timestring),
			ID:           fmt.Sprintf("ID-%d-@%s", i, timestring),
			ConnID:       connID,
			TimestampUTC: models.GetCurrentTimeEpochMs(),
			TTL:          3600,
		}); err == nil {
			getWork <- msg1
		}
		time.Sleep(2 * time.Second)

	}

	// see when server get it back?
	log.Println("----- in MAIN every one shutdown..")
	cancel()
	wm1.Close()
	wm2.Close()
	log.Println("----- in MAIN Waiting for every one to close up shop..")

	wg.Wait()
	log.Println("The END")
}