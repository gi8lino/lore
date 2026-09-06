package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// decode reads exactly one non-null, size-limited JSON value and rejects unknown fields.
func decode[T any](w http.ResponseWriter, r *http.Request) (T, error) {
	var zero T
	r.Body = http.MaxBytesReader(w, r.Body, 2<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	// A pointer distinguishes JSON null from the zero value of a request struct.
	var value *T
	if err := decoder.Decode(&value); err != nil {
		return zero, fmt.Errorf("invalid JSON request: %w", err)
	}
	if value == nil {
		return zero, errors.New("invalid JSON request: null is not allowed")
	}

	// Decode reads one value, not the complete body. Require EOF to reject a
	// second value, trailing junk, and oversized whitespace after valid JSON.
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		if err == nil {
			return zero, errors.New("invalid JSON request: only one value is allowed")
		}
		return zero, fmt.Errorf("invalid JSON request: %w", err)
	}
	return *value, nil
}

// jsonSlice keeps collection responses as arrays even when a service returns nil.
func jsonSlice[T any](values []T) []T {
	if values == nil {
		return []T{}
	}
	return values
}
