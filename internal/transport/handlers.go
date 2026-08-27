package transport

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"unicode/utf8"

	"go-image-service/internal/auth"
	"go-image-service/internal/metadata"
	"go-image-service/internal/upload"
)

type healthResponse struct {
	Status string `json:"status"`
}
type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}
type registerResponse struct {
	ID string `json:"id"`
}
type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}
type loginResponse struct {
	Token string `json:"token"`
}
type createUploadRequest struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
}
type createUploadResponse struct {
	ID        string `json:"id"`
	URL       string `json:"url"`
	ExpiresIn int    `json:"expires_in"`
}
type statusResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, healthResponse{Status: "ok"})
}

func (s *Server) register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	if req.Email == "" || req.Password == "" {
		http.Error(w, "email and password are required", http.StatusBadRequest)
		return
	}
	if utf8.RuneCountInString(req.Password) < 8 {
		http.Error(w, "password must be at least 8 characters", http.StatusBadRequest)
		return
	}

	id, err := s.auth.Register(r.Context(), req.Email, req.Password)
	if err != nil {
		http.Error(w, "could not register", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, registerResponse{ID: id})
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	token, err := s.auth.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			http.Error(w, "invalid email or password", http.StatusUnauthorized)
			return
		}
		http.Error(w, "could not log in", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, loginResponse{Token: token})
}

// createUpload negotiates a presigned upload: it records a pending row, starts
// the workflow, and returns a URL the client PUTs the bytes to directly.
func (s *Server) createUpload(w http.ResponseWriter, r *http.Request) {
	var req createUploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	if req.Filename == "" {
		http.Error(w, "filename is required", http.StatusBadRequest)
		return
	}
	switch req.ContentType {
	case "image/jpeg", "image/png":
	default:
		http.Error(w, "content_type must be image/jpeg or image/png", http.StatusBadRequest)
		return
	}

	userID, _ := UserIDFromContext(r.Context())
	id, url, err := s.uploads.Negotiate(r.Context(), req.Filename, req.ContentType, userID)
	if err != nil {
		log.Printf("createUpload: %v", err)
		http.Error(w, "could not create upload", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, createUploadResponse{
		ID:        id,
		URL:       url,
		ExpiresIn: upload.PresignTTLSeconds(),
	})
}

// completeUpload signals the workflow that the client finished uploading.
func (s *Server) completeUpload(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if len(id) != 32 {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	if _, _, ok := s.requireOwner(w, r, id); !ok {
		return
	}
	if err := s.uploads.Complete(r.Context(), id); err != nil {
		http.Error(w, "could not complete upload", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) images(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if len(id) != 32 {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	_, contentType, ok := s.requireOwner(w, r, id)
	if !ok {
		return
	}

	size := r.URL.Query().Get("size")
	switch size {
	case "original", "12x12", "25x25":
	default:
		http.Error(w, "Invalid or missing size", http.StatusBadRequest)
		return
	}

	data, err := s.objects.Get(r.Context(), id+"/"+size)
	if err != nil {
		http.Error(w, "Image not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Write(data)
}

func (s *Server) imageStatus(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if len(id) != 32 {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	st, _, ok := s.requireOwner(w, r, id)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, statusResponse{ID: id, Status: st})
}

// requireOwner looks up an upload and checks the caller owns it. It returns the
// status and content type only if the caller is the owner; otherwise it has
// already written a 404 (not 403, to avoid revealing an id exists) and returns
// ok=false.
func (s *Server) requireOwner(w http.ResponseWriter, r *http.Request, id string) (status, contentType string, ok bool) {
	st, ownerID, ct, err := s.reads.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, metadata.ErrNotFound) {
			http.Error(w, "Not found", http.StatusNotFound)
		} else {
			http.Error(w, "Server error", http.StatusInternalServerError)
		}
		return "", "", false
	}

	callerID, _ := UserIDFromContext(r.Context())
	if ownerID != callerID {
		http.Error(w, "Not found", http.StatusNotFound)
		return "", "", false
	}
	return st, ct, true
}
