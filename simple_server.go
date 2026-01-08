// Simple HTTP Server to serve the dashboard
// This is a minimal server for development purposes
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

func main() {
	// Get working directory
	workDir, _ := os.Getwd()
	staticDir := filepath.Join(workDir, "web", "static")

	// Serve static files
	fs := http.FileServer(http.Dir(staticDir))
	
	// Handle /static/ prefix
	http.Handle("/static/", http.StripPrefix("/static/", fs))
	
	// Handle root - serve index.html
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.ServeFile(w, r, filepath.Join(staticDir, "index.html"))
		} else {
			http.NotFound(w, r)
		}
	})

	port := ":8080"
	fmt.Printf("\n✅ Simple HTTP Server running on http://localhost%s\n", port)
	fmt.Printf("📂 Serving files from: %s\n", staticDir)
	fmt.Printf("👉 Open: http://localhost:8080/\n\n")
	fmt.Println("⚠️  Note: This is a static file server. API endpoints won't work until you run the full Go server.")
	
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("❌ Server failed: %v", err)
	}
}
