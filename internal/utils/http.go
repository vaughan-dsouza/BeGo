package utils

import (
	"encoding/json"
	"net/http"
)

// JSON writes a JSON response with status code.
func JSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if data != nil {
		_ = json.NewEncoder(w).Encode(data)
	}
}

// JSONError writes {"error": "..."} with a given status.
func JSONError(w http.ResponseWriter, status int, msg string) {
	JSON(w, status, map[string]string{"error": msg})
}

// DecodeJSON parses the JSON body into v and handles invalid JSON.
func DecodeJSON(w http.ResponseWriter, r *http.Request, v interface{}) error {
	if r.Body == nil {
		JSONError(w, http.StatusBadRequest, "empty request body")
		return http.ErrBodyNotAllowed
	}

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(v); err != nil {
		JSONError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return err
	}

	return nil
}
