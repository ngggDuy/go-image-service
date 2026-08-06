package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

// 10 MB maximum upload size
const maxUploadSize = 10 * 1024 * 1024

type HealthResponse struct {
	Status string `json:"status"`
}

type UploadResponse struct {
	ID string `json:"id"`
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(HealthResponse{Status: "ok"})
}

// Helper to generates a new 32 character unique ID from 16 random bytes
func newID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// uploadHandler
func uploadHandler(w http.ResponseWriter, r *http.Request) {
	// Limit upload file size
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)

	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			http.Error(w, "File too large (max 10 MB)", http.StatusRequestEntityTooLarge) // 413
			return
		}
		http.Error(w, "Malformed upload request", http.StatusBadRequest) //400
		return
	}

	// Get file from form field "image"
	// must match the name attribute in the form: <input type="file" name="image">
	file, _, err := r.FormFile("image")
	if err != nil {
		if errors.Is(err, http.ErrMissingFile) {
			http.Error(w, "No image file provided", http.StatusBadRequest)
			return
		}
		http.Error(w, "Error retrieving uploaded image", http.StatusBadRequest)
		return
	}
	defer file.Close() // Ensure multipart temp file resource is cleaned up

	// Validate file type
	buff := make([]byte, 512)
	if _, err := file.Read(buff); err != nil {
		http.Error(w, "Error reading uploaded file", http.StatusInternalServerError)
		return
	}
	fileType := http.DetectContentType(buff)

	var ext string
	switch fileType {
	case "image/jpeg":
		ext = ".jpg"
	case "image/png":
		ext = ".png"
	default:
		http.Error(w, "Invalid file type, only JPEG and PNG are supported.", http.StatusUnsupportedMediaType)
		return
	}

	// Reset read pointer to beginning of file after sniffing check
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		http.Error(w, "Error processing file", http.StatusInternalServerError)
		return
	}

	// Generate ID
	id, err := newID()
	if err != nil {
		http.Error(w, "Error when generating new image ID", http.StatusInternalServerError)
		return
	}
	dir := filepath.Join("uploads", id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		http.Error(w, "Error creating storage folder", http.StatusInternalServerError)
		return
	}

	// os.CreateTemp generates a random string replacing the `*`
	// e.g., "upload-1934857.jpg"
	dst, err := os.Create(filepath.Join(dir, "original"+ext))
	if err != nil {
		http.Error(w, "Error saving file", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	// Stream uploaded file to destination file
	if _, err := io.Copy(dst, file); err != nil {
		http.Error(w, "Error writing file", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(UploadResponse{ID: id})
}

func main() {
	// Create safe, unique file destination on disk before even starting server.
	// Upload directories don't change on request so initiating it inside handler is wasteful
	// Ensure uploads directory exists
	if err := os.MkdirAll("./uploads", 0o755); err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("POST /upload", uploadHandler)
	// mux.HandleFunc("GET /response", responseHandler)

	log.Fatal(http.ListenAndServe(":8080", mux))

}
