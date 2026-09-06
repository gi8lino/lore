package handler

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecodeRequestContract(t *testing.T) {
	t.Parallel()
	type request struct {
		Title string `json:"title"`
	}
	for _, tt := range []struct {
		name, body string
		valid      bool
	}{
		{"object", `{"title":"Example"}`, true},
		{"trailing whitespace", "{\"title\":\"Example\"} \n\t", true},
		{"empty body", "", false},
		{"null", "null", false},
		{"array", "[]", false},
		{"unknown field", `{"unknown":1}`, false},
		{"wrong field type", `{"title":42}`, false},
		{"second object", `{"title":"Example"}{}`, false},
		{"second null", `{"title":"Example"} null`, false},
		{"trailing junk", `{"title":"Example"}oops`, false},
		{"oversized value", `{"title":"` + strings.Repeat("a", 2<<20) + `"}`, false},
		{"oversized trailing whitespace", `{}` + strings.Repeat(" ", 2<<20), false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := decode[request](httptest.NewRecorder(), httptest.NewRequest("POST", "/", strings.NewReader(tt.body)))
			if (err == nil) != tt.valid {
				t.Fatalf("decode error = %v, want valid = %t", err, tt.valid)
			}
			if tt.valid && result.Title != "Example" {
				t.Fatalf("decoded title = %q", result.Title)
			}
		})
	}
}

func TestJSONSlice(t *testing.T) {
	t.Parallel()
	if got := jsonSlice[string](nil); got == nil || len(got) != 0 {
		t.Fatalf("nil collection became %#v", got)
	}
	values := []string{"one"}
	if got := jsonSlice(values); len(got) != 1 || got[0] != "one" {
		t.Fatalf("nonempty collection became %#v", got)
	}
}
