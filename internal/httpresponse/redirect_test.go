package httpresponse

import "testing"

func TestIsLocalPath(t *testing.T) {
	for _, value := range []string{"/", "/pages/start?search=hello#section", "/pages/a%20b"} {
		if !IsLocalPath(value) {
			t.Errorf("rejected local path %q", value)
		}
	}
	for _, value := range []string{"", "https://evil.example", "//evil.example", "/\\evil.example", "/\t/evil.example", "/\n/evil.example", "/\r/evil.example", "/\x00evil"} {
		if IsLocalPath(value) {
			t.Errorf("accepted unsafe redirect %q", value)
		}
	}
}
