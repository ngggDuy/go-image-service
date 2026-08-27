package transport

import (
	"context"
	"encoding/json"
	"net/http"
)

const maxUploadSize = 10 * 1024 * 1024 // 10 MB

// Authenticator handles registration, login, and token verification.
// The auth-service gRPC client (authclient.Client) satisfies it.
type Authenticator interface {
	Register(ctx context.Context, email, password string) (string, error)
	Login(ctx context.Context, email, password string) (string, error)
	Verify(ctx context.Context, token string) (string, error)
}

// Uploader is the upload use-case this layer calls. upload.Service satisfies it.
type Uploader interface {
	Process(ctx context.Context, data []byte, filename, ext, userID string) (string, error)
}

// ImageStore locates stored image files for serving. storage.Store satisfies it.
type ImageStore interface {
	Path(id, size string) (string, error)
}

// UploadReader reads an upload's status and owner. metadata.Repository satisfies it.
type UploadReader interface {
	Get(ctx context.Context, id string) (status, userID string, err error)
}

// Server holds the dependencies the HTTP handlers need and wires the routes.
type Server struct {
	auth    Authenticator
	uploads Uploader
	store   ImageStore
	reads   UploadReader
}

func New(auth Authenticator, uploads Uploader, store ImageStore, reads UploadReader) *Server {
	return &Server{auth: auth, uploads: uploads, store: store, reads: reads}
}

// Routes registers every route and returns the HTTP handler.
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	// Public endpoints.
	mux.HandleFunc("POST /register", s.register)
	mux.HandleFunc("POST /login", s.login)
	mux.HandleFunc("GET /health", s.health)

	// Protected: a valid Bearer token is required, and reads are owner-checked.
	mux.Handle("POST /upload", s.requireAuth(http.HandlerFunc(s.upload)))
	mux.Handle("GET /images/{id}", s.requireAuth(http.HandlerFunc(s.images)))
	mux.Handle("GET /images/{id}/status", s.requireAuth(http.HandlerFunc(s.imageStatus)))

	return mux
}

// writeJSON writes v as a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
