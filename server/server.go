package server

import (
	"bufio"
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

var (
	logFilePath   = "../logs.txt"
	logChannel    chan Log
	flushMutex    sync.Mutex
	logFile       *os.File
	DB            *sql.DB
	flushInterval = 30 * time.Second
)

// Log represents the log structure
type Log struct {
	ID        int    `json:"id"`
	UnixTS    int64  `json:"unix_ts"`
	UserID    int    `json:"user_id"`
	EventName string `json:"event_name"`
}

func LogHandler(w http.ResponseWriter, r *http.Request) {
	var logData Log
	err := json.NewDecoder(r.Body).Decode(&logData)
	if err != nil {
		log.Println("Failed to decode log data:", err)
		http.Error(w, "Failed to decode log data", http.StatusBadRequest)
		return
	}

	logChannel <- logData

	w.WriteHeader(http.StatusOK)
}

func writeLogs() {
	for logData := range logChannel {

		logBytes, err := json.Marshal(logData)
		if err != nil {
			log.Println("Failed to marshal log data:", err)
			continue
		}

		_, err = logFile.Write(append(logBytes, '\n'))
		if err != nil {
			log.Println("Failed to write log to file:", err)
			continue
		}
		err = logFile.Sync()
		if err != nil {
			log.Println("Failed to sync writing log to file:", err)
			continue
		}
	}
}

func FlushLogs() {
	flushMutex.Lock()
	defer flushMutex.Unlock()

	file, err := os.Open(logFilePath)
	if err != nil {
		log.Println("Failed to open log file for reading:", err)
		return
	}
	defer file.Close()

	// Start a transaction
	tx, err := DB.Begin()
	if err != nil {
		log.Println("Failed to start database transaction:", err)
		return
	}

	stmt, err := tx.Prepare("INSERT INTO logs (unix_ts, user_id, event_name) VALUES ($1, $2, $3)")
	if err != nil {
		log.Println("Failed to prepare SQL statement:", err)
		tx.Rollback()
		return
	}
	defer stmt.Close()

	// Read logs from the file and insert into the database
	scanner := bufio.NewScanner(file)
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		var logData Log
		err := json.Unmarshal(bytes.Trim(scanner.Bytes(), "\x00"), &logData)
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

func InitDB() {
	host := os.Getenv("POSTGRES_HOST")
	port := 5432
	user := os.Getenv("POSTGRES_USER")
	password := os.Getenv("POSTGRES_PASSWORD")
	dbname := os.Getenv("POSTGRES_DB")

	connectionString := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)

	var err error
	DB, err = sql.Open("postgres", connectionString)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	driver, err := postgres.WithInstance(DB, &postgres.Config{})
	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		dbname, driver)

	err = m.Up()

	if err != nil {
		log.Println("Creating logs table:", err)
	}

	log.Println("Database connection established.")
}

func init() {
	// Initialize the log writer
	logChannel = make(chan Log, 15000)
	go writeLogs()

	var err error
	logFile, err = os.Create(logFilePath)
	if err != nil {
		log.Fatal("Failed to open log file:", err)
	}

	// Start the flush ticker
	go func() {
		for range time.Tick(flushInterval) {
			FlushLogs()
		}
	}()
}
