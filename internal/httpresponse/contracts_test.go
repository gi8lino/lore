package httpresponse

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"
)

// These fixtures are also consumed by the TypeScript HTTP contract tests.
func TestProblemSharedContract(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile("../../test/contracts/http.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixtures map[string]json.RawMessage
	if err := json.Unmarshal(data, &fixtures); err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name, message string
		fields        []FieldProblem
	}{
		{"problem_without_fields", "Page not found.", nil},
		{"problem_with_fields", "Page validation failed.", []FieldProblem{NewFieldProblem("title", "Title is required.")}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			response := httptest.NewRecorder()
			Problem(response, http.StatusBadRequest, tt.message, tt.fields...)
			var got, want any
			if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(fixtures[tt.name], &want); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("response = %#v, want %#v", got, want)
			}
		})
	}
}
