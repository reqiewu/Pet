package transport

import (
	"encoding/json"
	"net/http"
)

type JSONResponder struct{}

func (r *JSONResponder) ErrorJSON(w http.ResponseWriter, message string, status int) {
	r.WriteJSON(w, status, map[string]string{"error": message})
}

func (r *JSONResponder) WriteJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
