package main

import (
	"fable-assignment/server"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/joho/godotenv/autoload"
	_ "github.com/lib/pq"
	"log"
	"net/http"
	"os"
	"os/signal"
)

var (
	shutdownChan chan os.Signal
)

func main() {
	shutdownChan = make(chan os.Signal, 1)

	// Initialize the database
	server.InitDB()

	// Register the HTTP handler
	http.HandleFunc("/log", server.LogHandler)

	signal.Notify(shutdownChan, os.Interrupt)
	go func() {
		for _ = range shutdownChan {
			// Flush any remaining logs
			server.FlushLogs()

			// Close the database connection
			server.DB.Close()

			log.Println("Server stopped.")
			os.Exit(0)
		}
	}()

	log.Fatal(http.ListenAndServe(":9000", nil))
}
