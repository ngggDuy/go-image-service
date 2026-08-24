package transport

import (
	"errors"
	"io"
	"net/http"

	"go-image-service/internal/metadata"
	"go-image-service/internal/storage"
)

type healthResponse struct {
	Status string `json:"status"`
}

type uploadResponse struct {
	ID string `json:"id"`
}

type statusResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, healthResponse{Status: "ok"})
}

func (s *Server) upload(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)

	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			http.Error(w, "File too large (max 10 MB)", http.StatusRequestEntityTooLarge)
			return
		}
		http.Error(w, "Malformed upload request", http.StatusBadRequest)
		return
	}

	file, fileHeader, err := r.FormFile("image")
	if err != nil {
		if errors.Is(err, http.ErrMissingFile) {
			http.Error(w, "No image file provided", http.StatusBadRequest)
			return
		}
		http.Error(w, "Error retrieving uploaded image", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Sniff the content type from the first 512 bytes.
	buff := make([]byte, 512)
	if _, err := file.Read(buff); err != nil {
		http.Error(w, "Error reading uploaded file", http.StatusInternalServerError)
		return
	}
	var ext string
	switch http.DetectContentType(buff) {
	case "image/jpeg":
		ext = ".jpg"
	case "image/png":
		ext = ".png"
	default:
		http.Error(w, "Invalid file type, only JPEG and PNG are supported.", http.StatusUnsupportedMediaType)
		return
	}

	// Rewind after sniffing, then read the whole file into memory.
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		http.Error(w, "Error processing file", http.StatusInternalServerError)
		return
	}
	data, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Error reading the uploaded file", http.StatusInternalServerError)
		return
	}

	// Process stores the original + starts the async resize workflow, then returns.
	id, err := s.uploads.Process(r.Context(), data, fileHeader.Filename, ext)
	if err != nil {
		http.Error(w, "Error processing upload", http.StatusInternalServerError)
		return
	}

	// 202 Accepted: work is underway; poll GET /images/{id}/status for completion.
	writeJSON(w, http.StatusAccepted, uploadResponse{ID: id})
}

func (s *Server) images(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if len(id) != 32 {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	size := r.URL.Query().Get("size")
	switch size {
	case "original", "12x12", "25x25":
	default:
		http.Error(w, "Invalid or missing size", http.StatusBadRequest)
		return
	}

	path, err := s.store.Path(id, size)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			http.Error(w, "Image not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	http.ServeFile(w, r, path)
}

func (s *Server) imageStatus(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if len(id) != 32 {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	st, err := s.status.GetStatus(r.Context(), id)
	if err != nil {
		if errors.Is(err, metadata.ErrNotFound) {
			http.Error(w, "Upload not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, statusResponse{ID: id, Status: st})
}
