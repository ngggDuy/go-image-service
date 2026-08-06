package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"go-image-service/gen/imageprocess"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// 10 MB maximum upload size
const maxUploadSize = 10 * 1024 * 1024

// Handlers are currently package level funcs so can't reach clients.
// Fix: we add a small struct that holds dependencies to inject into the client.
type app struct {
	resizer imageprocess.ResizerClient
}

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
func (a *app) uploadHandler(w http.ResponseWriter, r *http.Request) {
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

	// read image into memory (< 10 MB) so we can store it and send to resizer process
	data, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Error reading the uploaded file", http.StatusInternalServerError)
		return
	}

	// save original
	if err := os.WriteFile(filepath.Join(dir, "original"+ext), data, 0o644); err != nil {
		http.Error(w, "Error saving original", http.StatusInternalServerError)
		return
	}

	// resize image to each size by calling image service over gRPC, save result
	sizes := []struct {
		name string
		w, h int32
	}{{"12x12", 12, 12}, {"25x25", 25, 25}}

	for _, size := range sizes {
		resp, err := a.resizer.Resize(r.Context(), &imageprocess.ResizeRequest{
			ImageToResize: data,
			ImageWidth:    size.w,
			ImageHeight:   size.h,
		})
		if err != nil {
			log.Printf("resize RPC failed: %v", err) // logged here to find bug where gRPC limits request to 4MB by default to prevent OOM
			http.Error(w, "Error resizing image", http.StatusInternalServerError)
			return
		}
		if err := os.WriteFile(filepath.Join(dir, size.name+ext), resp.GetResizedImage(), 0o644); err != nil {
			http.Error(w, "Error saving resized image", http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(UploadResponse{ID: id})
}

func main() {
	// create gRPC client and build app
	conn, err := grpc.NewClient(
		"localhost:50051",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	a := &app{resizer: imageprocess.NewResizerClient(conn)}

	// Create safe, unique file destination on disk before even starting server.
	// Upload directories don't change on request so initiating it inside handler is wasteful
	// Ensure uploads directory exists
	if err := os.MkdirAll("./uploads", 0o755); err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("POST /upload", a.uploadHandler)
	// mux.HandleFunc("GET /response", responseHandler)

	log.Fatal(http.ListenAndServe(":8080", mux))

}
