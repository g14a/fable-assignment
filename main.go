package main

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	_ "github.com/lib/pq"
)

// Log represents the log structure
type Log struct {
	ID        int    `json:"id"`
	UnixTS    int64  `json:"unix_ts"`
	UserID    int    `json:"user_id"`
	EventName string `json:"event_name"`
}

var (
	db            *sql.DB
	flushInterval = 30 * time.Second
	shutdownChan  chan os.Signal
	logFilePath   = "logs.txt"
	logFileMutex  sync.Mutex
)

const (
	maxLogFileSize = 10 * 1024 * 1024 // 1MB
)

func main() {
	shutdownChan = make(chan os.Signal, 1)

	// Initialize the database
	initDB()

	// Register the HTTP handler
	http.HandleFunc("/log", handleLog)

	signal.Notify(shutdownChan, os.Interrupt)
	go func() {
		for _ = range shutdownChan {
			// Flush any remaining logs
			flushLogs()

			// Close the database connection
			db.Close()

			log.Println("Server stopped.")
			os.Exit(0)
		}
	}()

	log.Fatal(http.ListenAndServe(":8080", nil))
}

func initDB() {
	// Replace with your PostgreSQL connection details
	host := "postgres"
	port := 5432
	user := os.Getenv("POSTGRES_USER")
	password := os.Getenv("POSTGRES_PASSWORD")
	dbname := os.Getenv("POSTGRES_DB")

	connectionString := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)

	var err error
	db, err = sql.Open("postgres", connectionString)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		dbname, driver)

	err = m.Up()

	if err != nil {
		log.Println("Creating logs table:", err)
	}

	log.Println("Database connection established.")
}

var logFile *os.File

func init() {
	// Open the log file in append mode
	var err error
	logFile, err = os.Create(logFilePath)
	if err != nil {
		log.Fatal("Failed to open log file:", err)
	}

	// Start the flush ticker
	go func() {
		for range time.Tick(flushInterval) {
			flushLogsIfRequired()
		}
	}()
}

func flushLogsIfRequired() {
	fileInfo, err := logFile.Stat()
	if err != nil {
		log.Println("Failed to get log file info:", err)
		return
	}

	fileSize := fileInfo.Size()

	if fileSize > maxLogFileSize {
		flushLogs()

		// Rewind the log file to the beginning
		_, err := logFile.Seek(0, 0)
		if err != nil {
			log.Println("Failed to seek to the beginning of the log file:", err)
			return
		}

		// Truncate the file by writing an empty byte slice
		_, err = logFile.Write(make([]byte, 0))
		if err != nil {
			log.Println("Failed to clear log file:", err)
			return
		}
	}
}

func handleLog(w http.ResponseWriter, r *http.Request) {
	var logData Log
	err := json.NewDecoder(r.Body).Decode(&logData)
	if err != nil {
		log.Println("Failed to decode log data:", err)
		http.Error(w, "Failed to decode log data", http.StatusBadRequest)
		return
	}

	// Acquire the lock to ensure exclusive access to the log file
	logFileMutex.Lock()
	defer logFileMutex.Unlock()

	// Write the log to the file
	logBytes, err := json.Marshal(logData)
	if err != nil {
		log.Println("Failed to marshal log data:", err)
		http.Error(w, "Failed to marshal log data", http.StatusInternalServerError)
		return
	}

	_, err = logFile.Write(append(logBytes, '\n'))
	if err != nil {
		log.Println("Failed to write log to file:", err)
		http.Error(w, "Failed to write log to file", http.StatusInternalServerError)
		return
	}

	// Send response
	w.WriteHeader(http.StatusOK)
}

func flushLogs() {
	// Acquire the lock to ensure exclusive access to the log file
	logFileMutex.Lock()
	defer logFileMutex.Unlock()

	// Open the log file for reading
	file, err := os.Open(logFilePath)
	if err != nil {
		log.Println("Failed to open log file for reading:", err)
		return
	}
	defer file.Close()

	// Start a transaction
	tx, err := db.Begin()
	if err != nil {
		log.Println("Failed to start database transaction:", err)
		return
	}

	// Prepare the SQL statement
	stmt, err := tx.Prepare("INSERT INTO logs (unix_ts, user_id, event_name) VALUES ($1, $2, $3)")
	if err != nil {
		log.Println("Failed to prepare SQL statement:", err)
		tx.Rollback()
		return
	}
	defer stmt.Close()

	// Read logs from the file and insert into the database
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var logData Log
		err := json.Unmarshal(scanner.Bytes(), &logData)
		if err != nil {
			log.Println("Failed to unmarshal log data:", err)
			continue
		}
		_, err = stmt.Exec(logData.UnixTS, logData.UserID, logData.EventName)
		if err != nil {
			log.Println("Failed to insert log:", err)
			tx.Rollback()
			return
		}
	}

	if err := scanner.Err(); err != nil {
		log.Println("Error occurred while reading log file:", err)
		tx.Rollback()
		return
	}

	// Commit the transaction
	err = tx.Commit()
	if err != nil {
		log.Println("Failed to commit database transaction:", err)
		tx.Rollback()
		return
	}

	log.Println("Logs flushed to database.")
}
