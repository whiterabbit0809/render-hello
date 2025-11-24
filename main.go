package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	// Serve static files from ./static (index.html, CSS, JS, etc.)
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)

	// Simple JSON API to prove it’s working
	http.HandleFunc("/api/hello", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"message":"Hello from Go on Render!"}`))
	})

	// Render will set PORT; default to 3000 for local dev
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	log.Printf("Server listening on port %s\n", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
