package transport

import (
	"context"
	"encoding/json"
	"net/http"
)

// Authenticator handles registration, login, and token verification.
// The auth-service gRPC client (authclient.Client) satisfies it.
type Authenticator interface {
	Register(ctx context.Context, email, password string) (string, error)
	Login(ctx context.Context, email, password string) (string, error)
	Verify(ctx context.Context, token string) (string, error)
}

// Uploads negotiates presigned uploads and forwards completion. upload.Service satisfies it.
type Uploads interface {
	Negotiate(ctx context.Context, filename, contentType, userID string) (id, url string, err error)
	Complete(ctx context.Context, id string) error
}

// UploadReader reads an upload's status, owner, and content type. metadata.Repository satisfies it.
type UploadReader interface {
	Get(ctx context.Context, id string) (status, userID, contentType string, err error)
}

// ObjectGetter fetches stored object bytes for serving. storage.MinioStore satisfies it.
type ObjectGetter interface {
	Get(ctx context.Context, key string) ([]byte, error)
}

// Server holds the dependencies the HTTP handlers need and wires the routes.
type Server struct {
	auth    Authenticator
	uploads Uploads
	reads   UploadReader
	objects ObjectGetter
}

func New(auth Authenticator, uploads Uploads, reads UploadReader, objects ObjectGetter) *Server {
	return &Server{auth: auth, uploads: uploads, reads: reads, objects: objects}
}

// Routes registers every route and returns the HTTP handler.
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	// Public endpoints.
	mux.HandleFunc("POST /register", s.register)
	mux.HandleFunc("POST /login", s.login)
	mux.HandleFunc("GET /health", s.health)

	// Protected: a valid Bearer token is required; reads are owner-checked.
	mux.Handle("POST /uploads", s.requireAuth(http.HandlerFunc(s.createUpload)))
	mux.Handle("POST /uploads/{id}/complete", s.requireAuth(http.HandlerFunc(s.completeUpload)))
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
