package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	_ "github.com/joho/godotenv/autoload"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

const (
	NumRequestsPerBatch = 10000
	NumBatches          = 10
	RequestDelay        = 100 * time.Millisecond
)

var URL = "http://" + os.Getenv("APP_HOST") + ":9000/log"

type TestLog struct {
	ID        int    `json:"id"`
	UnixTS    int64  `json:"unix_ts"`
	UserID    int    `json:"user_id"`
	EventName string `json:"event_name"`
}

func main() {
	start := time.Now()
	var wg sync.WaitGroup

	for i := 1; i <= NumBatches; i++ {
		for j := 1; j <= NumRequestsPerBatch; j++ {
			wg.Add(1)
			go sendLog(j, &wg)
			time.Sleep(time.Millisecond)
		}
		fmt.Println("Sent ", i*NumRequestsPerBatch, " Requests")
		time.Sleep(RequestDelay)
	}

	wg.Wait()
	elapsed := time.Since(start)
	fmt.Printf("Sent %d logs in %s\n", NumRequestsPerBatch*NumBatches, elapsed)
}

func sendLog(i int, wg *sync.WaitGroup) {
	defer wg.Done()

	logData := TestLog{
		ID:        i,
		UnixTS:    time.Now().Unix(),
		UserID:    i,
		EventName: "login",
	}

	logBytes, err := json.Marshal(logData)
	if err != nil {
		log.Println("Failed to marshal log data:", err)
		return
	}

	resp, err := http.Post(URL, "application/json", bytes.NewBuffer(logBytes))
	if err != nil {
		log.Println("Failed to send log:", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Failed to send log. Status: %s\n", resp.Status)
		return
	}
}
