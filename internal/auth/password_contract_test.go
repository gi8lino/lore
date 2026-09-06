package auth

import (
	"encoding/json"
	"os"
	"testing"
)

func TestLocalPasswordSharedContract(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile("../../test/contracts/passwords.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct{ Name, Password, Problem string }
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			t.Parallel()
			if got := LocalPasswordProblem(tt.Password); got != tt.Problem {
				t.Fatalf("problem = %q, want %q", got, tt.Problem)
			}
			if ValidLocalPassword(tt.Password) != (tt.Problem == "") {
				t.Fatal("validation predicate disagrees with problem")
			}
		})
	}
}

func TestLocalPasswordRejectsInvalidUTF8(t *testing.T) {
	t.Parallel()
	if ValidLocalPassword("valid-length-" + string([]byte{0xff})) {
		t.Fatal("invalid UTF-8 was accepted")
	}
}
