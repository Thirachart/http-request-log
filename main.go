package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
)

type RequestLog struct {
	ID        int       `json:"id"`
	Method    string    `json:"method"`
	URL       string    `json:"url"`
	Headers   string    `json:"headers"`
	Body      string    `json:"body"`
	Timestamp time.Time `json:"timestamp"`
}

var db *sql.DB

func initDB() {
	var err error
	db, err = sql.Open("sqlite3", "./sql_app.db")
	if err != nil {
		log.Fatal(err)
	}

	createTableSQL := `CREATE TABLE IF NOT EXISTS request_logs (
		"id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,		
		"method" TEXT,
		"url" TEXT,
		"headers" TEXT,
		"body" TEXT,
		"timestamp" DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	_, err = db.Exec(createTableSQL)
	if err != nil {
		log.Fatal(err)
	}
}

func ingestHandler(w http.ResponseWriter, r *http.Request) {
	// Read body
	var bodyBytes []byte
	if r.Body != nil {
		var err error
		bodyBytes, err = io.ReadAll(r.Body)
		if err != nil {
			log.Printf("Error reading body: %v", err)
			http.Error(w, "Failed to read body", http.StatusInternalServerError)
			return
		}
		r.Body.Close()
	}

	// Get headers
	headers, _ := json.Marshal(r.Header)

	// Log to DB
	_, err := db.Exec("INSERT INTO request_logs (method, url, headers, body, timestamp) VALUES (?, ?, ?, ?, ?)",
		r.Method, r.URL.String(), string(headers), string(bodyBytes), time.Now())
	if err != nil {
		log.Printf("Error logging request: %v", err)
		http.Error(w, "Failed to log request", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{
		"status":    "captured",
		"timestamp": time.Now().Format(time.RFC3339),
		"size":      fmt.Sprintf("%d bytes", len(bodyBytes)),
	})
}

func getLogsHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, method, url, headers, body, timestamp FROM request_logs ORDER BY timestamp DESC LIMIT 50")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var logs []RequestLog
	for rows.Next() {
		var l RequestLog
		err := rows.Scan(&l.ID, &l.Method, &l.URL, &l.Headers, &l.Body, &l.Timestamp)
		if err != nil {
			log.Printf("Scan error: %v", err)
			continue
		}
		logs = append(logs, l)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}

func main() {
	godotenv.Load()
	initDB()
	defer db.Close()

	mux := http.NewServeMux()
	
	// Separate Endpoints
	mux.HandleFunc("/admin/logs", getLogsHandler) // Endpoint สำหรับเรียกดู
	mux.HandleFunc("/ingest/", ingestHandler)    // Endpoint สำหรับเก็บ Log (เช่น /ingest/my-service)
	
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("HTTP Logger: Use /ingest/ to log and /admin/logs to view."))
	})

	fmt.Println("Server starting on :8000")
	log.Fatal(http.ListenAndServe(":8000", mux))
}
