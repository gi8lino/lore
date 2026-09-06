package markdown

import (
	"encoding/json"
	"os"
	"testing"
)

func TestSlugContract(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile("../../test/contracts/slugs.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct{ Name, Value, Slug string }
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			t.Parallel()
			if got := Slug(tt.Value); got != tt.Slug {
				t.Fatalf("Slug(%q) = %q, want %q", tt.Value, got, tt.Slug)
			}
		})
	}
}
