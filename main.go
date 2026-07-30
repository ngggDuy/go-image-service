package main

import (
	"encoding/json"
	"log"
	"net/http"
)

type HealthResponse struct {
	Status string `json:"status"`
}

func main() {
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(HealthResponse{Status: "ok"})
	})

	log.Fatal(http.ListenAndServe(":8080", nil))

}
