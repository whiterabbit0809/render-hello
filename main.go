package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/lib/pq"
)

var db *sql.DB

type resultResponse struct {
	Value float64 `json:"value"`
}

type addRequest struct {
	A float64 `json:"a"`
}

func main() {
	// --- DB setup ---

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL env var is not set")
	}

	var err error
	db, err = sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("error opening DB: %v", err)
	}

	// small, safe pool settings
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("error pinging DB: %v", err)
	}

	if err := initCalculatorTable(ctx, db); err != nil {
		log.Fatalf("error initializing calculator table: %v", err)
	}

	// --- HTTP routes ---

	// Serve static files from ./static (index.html, CSS, JS, etc.)
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)

	// Existing hello endpoint
	http.HandleFunc("/api/hello", helloHandler)

	// Calculator endpoints
	http.HandleFunc("/api/result", resultHandler)
	http.HandleFunc("/api/add", addHandler)

	// Port (Render injects PORT, local uses 3000)
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	log.Printf("Server listening on port %s\n", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

// Create table + default row (id=1, value=0) if missing
func initCalculatorTable(ctx context.Context, db *sql.DB) error {
	createTableSQL := `
CREATE TABLE IF NOT EXISTS calculator_state (
    id    integer PRIMARY KEY,
    value double precision NOT NULL
);`

	if _, err := db.ExecContext(ctx, createTableSQL); err != nil {
		return err
	}

	insertDefaultRowSQL := `
INSERT INTO calculator_state (id, value)
VALUES (1, 0)
ON CONFLICT (id) DO NOTHING;`

	if _, err := db.ExecContext(ctx, insertDefaultRowSQL); err != nil {
		return err
	}

	return nil
}

// --- Handlers ---

func helloHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]string{
		"message": "Hello from Go on Render!",
	})
}

func resultHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	value, err := getCurrentValue(r.Context())
	if err != nil {
		log.Printf("error getting current value: %v", err)
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, resultResponse{Value: value})
}

func addHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req addRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	newValue, err := addToValue(r.Context(), req.A)
	if err != nil {
		log.Printf("error adding to value: %v", err)
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, resultResponse{Value: newValue})
}

// --- DB helpers ---

func getCurrentValue(ctx context.Context) (float64, error) {
	var value float64
	err := db.QueryRowContext(ctx, `SELECT value FROM calculator_state WHERE id = 1`).Scan(&value)
	if err == sql.ErrNoRows {
		// Shouldn't happen because we insert a default row, but fall back to 0
		return 0, nil
	}
	return value, err
}

func addToValue(ctx context.Context, a float64) (float64, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}

	var current float64
	if err := tx.QueryRowContext(ctx, `SELECT value FROM calculator_state WHERE id = 1 FOR UPDATE`).Scan(&current); err != nil {
		_ = tx.Rollback()
		return 0, err
	}

	newValue := current + a

	if _, err := tx.ExecContext(ctx, `UPDATE calculator_state SET value = $1 WHERE id = 1`, newValue); err != nil {
		_ = tx.Rollback()
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return newValue, nil
}

// --- misc helpers ---

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, "failed to encode JSON", http.StatusInternalServerError)
	}
}
