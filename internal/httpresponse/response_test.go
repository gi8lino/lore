package httpresponse

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestProblemWritesStructuredJSON verifies the shared error contract used by handlers and middleware.
func TestProblemWritesStructuredJSON(t *testing.T) {
	t.Parallel()

	t.Run("writes field problems as an object", func(t *testing.T) {
		t.Parallel()

		response := httptest.NewRecorder()

		Problem(
			response,
			http.StatusUnprocessableEntity,
			"Page validation failed.",
			NewFieldProblem("slug", "A page path could not be derived."),
			NewFieldProblem("title", "Title is required."),
		)

		assert.Equal(t, http.StatusUnprocessableEntity, response.Code)
		assert.Equal(t, "application/json; charset=utf-8", response.Header().Get("Content-Type"))
		assert.JSONEq(t, `{
			"error": "Page validation failed.",
			"problems": {
				"slug": "A page path could not be derived.",
				"title": "Title is required."
			}
		}`, response.Body.String())
	})

	t.Run("keeps absent field problems null", func(t *testing.T) {
		t.Parallel()

		response := httptest.NewRecorder()

		Problem(response, http.StatusNotFound, "Page not found.")

		assert.Equal(t, http.StatusNotFound, response.Code)
		assert.JSONEq(t, `{
			"error": "Page not found.",
			"problems": null
		}`, response.Body.String())
	})
}
