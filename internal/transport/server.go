package transport

import (
	"context"
	"encoding/json"
	"net/http"
)

const maxUploadSize = 10 * 1024 * 1024 // 10 MB

// Uploader is the upload use-case this layer calls. upload.Service satisfies it.
type Uploader interface {
	Process(ctx context.Context, data []byte, filename, ext string) (string, error)
}

// ImageStore locates stored image files for serving. storage.Store satisfies it.
type ImageStore interface {
	Path(id, size string) (string, error)
}

// StatusReader reads an upload's processing status. metadata.Repository satisfies it.
type StatusReader interface {
	GetStatus(ctx context.Context, id string) (string, error)
}

// Server holds the dependencies the HTTP handlers need and wires the routes.
type Server struct {
	uploads Uploader
	store   ImageStore
	status  StatusReader
}

func New(uploads Uploader, store ImageStore, status StatusReader) *Server {
	return &Server{uploads: uploads, store: store, status: status}
}

// Routes registers every route and returns the HTTP handler.
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.health)
	mux.HandleFunc("POST /upload", s.upload)
	mux.HandleFunc("GET /images/{id}", s.images)
	mux.HandleFunc("GET /images/{id}/status", s.imageStatus)
	return mux
}

// writeJSON writes v as a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
