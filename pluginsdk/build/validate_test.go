package build

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidationRejectsUnsafeProjects(t *testing.T) {
	directory := t.TempDir()
	manifest := "api_version: 1\nid: com.example.test\nname: Test\nversion: 1.0.0\nmodules:\n  - type: renderer-extension\n    id: test\n    stage: preprocess\npermissions: []\n"
	write := func(name, value string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(directory, name), []byte(value), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write("plugin.yaml", manifest)
	write("README.md", "# Test")
	write("go.mod", "module example.com/test\n\ngo 1.27.0\n")
	if _, err := Validate(directory); err != nil {
		t.Fatal(err)
	}
	write("plugin.yaml", manifest+"unknown: value\n")
	if _, err := Validate(directory); err == nil {
		t.Fatal("unknown manifest field accepted")
	}
	write("plugin.yaml", strings.Replace(manifest, "api_version: 1", "api_version: 999", 1))
	if _, err := Validate(directory); err == nil {
		t.Fatal("incompatible version accepted")
	}
	write("plugin.yaml", manifest)
	if err := os.Mkdir(filepath.Join(directory, "assets"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(directory, "README.md"), filepath.Join(directory, "assets", "escape")); err != nil {
		t.Fatal(err)
	}
	if _, err := Validate(directory); err == nil {
		t.Fatal("asset symlink accepted")
	}
}
