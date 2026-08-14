package main

import (
	"context"
	"encoding/json"
	"errors"
	"go-image-service/gen/imageprocess"
	"go-image-service/internal/id"
	"go-image-service/internal/resizer"
	"go-image-service/internal/storage"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// 10 MB maximum upload size
const maxUploadSize = 10 * 1024 * 1024

// Handlers are currently package level funcs so can't reach clients.
// Fix: we add a small struct that holds dependencies to inject into the client.
type app struct {
	store   *storage.Store
	resizer *resizer.Client
	db      *pgxpool.Pool
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
	file, fileHeader, err := r.FormFile("image")
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
	id, err := id.New()
	if err != nil {
		http.Error(w, "Error when generating new image ID", http.StatusInternalServerError)
		return
	}

	// read image into memory (< 10 MB) so we can store it and send to resizer process
	data, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Error reading the uploaded file", http.StatusInternalServerError)
		return
	}

	// save original
	err = a.store.Save(id, "original", ext, data)
	if err != nil {
		http.Error(w, "Error saving original image", http.StatusInternalServerError)
		return
	}

	// resize image to each size by calling image service over gRPC, save result
	sizes := []struct {
		name string
		w, h int32
	}{{"12x12", 12, 12}, {"25x25", 25, 25}}

	for _, size := range sizes {
		log.Printf("upload %s: calling image service to resize %s", id, size.name)
		resized, err := a.resizer.Resize(r.Context(), data, int(size.w), int(size.h))
		if err != nil {
			log.Printf("resize RPC failed: %v", err)
			http.Error(w, "Error resizing image", http.StatusInternalServerError)
			return
		}
		// save resized versions
		err = a.store.Save(id, size.name, ext, resized)
		if err != nil {
			http.Error(w, "Error saving resized image", http.StatusInternalServerError)
			return
		}
	}

	// Save metadata via parameterized placeholders.
	// db driver sends values apart from query string to
	// prevent against SQL injection
	_, err = a.db.Exec(r.Context(),
		`INSERT INTO uploads (id, original_filename, ext, status)
       VALUES ($1, $2, $3, $4)`,
		id, fileHeader.Filename, ext, "complete",
	)
	if err != nil {
		http.Error(w, "Error saving metadata", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(UploadResponse{ID: id})
}

func (a *app) imagesHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if len(id) != 32 {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	size := r.URL.Query().Get("size")
	switch size {
	case "original", "12x12", "25x25": // ok
	default:
		http.Error(w, "Invalid or missing size", http.StatusBadRequest)
		return
	}

	path, err := a.store.Path(id, size)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			http.Error(w, "Image not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	http.ServeFile(w, r, path) // sets Content-Type + streams the bytes
}

func main() {
	// Create address for Image Service container
	imgServiceAddr := os.Getenv("IMAGE_SERVICE_ADDR")
	if imgServiceAddr == "" {
		imgServiceAddr = "localhost:50051" // local default
	}

	// create gRPC client and build app
	conn, err := grpc.NewClient(
		imgServiceAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	// connect to database
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://imageservice:secret@localhost:5432/imageservice"
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	// pgxpool.New is lazy connect so we pool.Ping at startup to fail fast if db is unreachable
	if err := pool.Ping(context.Background()); err != nil {
		log.Fatalf("Cannot reach postgres: %v", err)
	}

	// Create safe, unique file destination on disk before even starting server.
	// Upload directories don't change on request so initiating it inside handler is wasteful
	// Ensure uploads directory exists
	store, err := storage.New("uploads")
	if err != nil {
		log.Fatal(err)
	}

	a := &app{
		store:   store,
		resizer: resizer.New(imageprocess.NewResizerClient(conn)),
		db:      pool,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("POST /upload", a.uploadHandler)
	mux.HandleFunc("GET /images/{id}", a.imagesHandler)

	log.Fatal(http.ListenAndServe(":8080", mux))

}
