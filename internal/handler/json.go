package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// decode reads a size-limited JSON request and rejects unknown fields.
func decode[T any](w http.ResponseWriter, r *http.Request) (T, error) {
	var value T
	r.Body = http.MaxBytesReader(w, r.Body, 2<<20)
	decoder := json.NewDecoder(r.Body)

	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		return value, fmt.Errorf("invalid JSON request: %w", err)
	}

	return value, nil
}
