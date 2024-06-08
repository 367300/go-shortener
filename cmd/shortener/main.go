package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
)

var (
	mu    sync.Mutex
	store = make(map[string]string)
)

func generateID(store map[string]string) string {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		log.Fatal(err)
	}
	id := base64.URLEncoding.EncodeToString(b)
	id = strings.ReplaceAll(id, "=", "c")
	id = strings.ReplaceAll(id, "_", "D")
	id = strings.ReplaceAll(id, "-", "G")

	if _, exists := store[id]; exists {
		return generateID(store)
	}

	return id
}

func shortenURL(w http.ResponseWriter, r *http.Request) {

	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil || len(body) == 0 {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	originalURL := string(body)
	if !strings.HasPrefix(originalURL, "http://") && !strings.HasPrefix(originalURL, "https://") {
		http.Error(w, "Invalid URL format", http.StatusBadRequest)
		return
	}

	mu.Lock()
	id := generateID(store)
	mu.Unlock()

	store[id] = originalURL

	shortenedURL := fmt.Sprintf("http://localhost:8080/%s", id)
	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(shortenedURL))
}

func redirectToOriginalURL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusBadRequest)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/")

	mu.Lock()
	originalURL, exists := store[id]
	mu.Unlock()

	if !exists {
		http.Error(w, "URL not found", http.StatusBadRequest)
		return
	}

	http.Redirect(w, r, originalURL, http.StatusTemporaryRedirect)
	w.Header().Set("Location", originalURL)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			shortenURL(w, r)
		} else if r.Method == http.MethodGet {
			redirectToOriginalURL(w, r)
		} else {
			http.Error(w, "Invalid request method", http.StatusBadRequest)
		}
	})

	fmt.Println("Server is running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
