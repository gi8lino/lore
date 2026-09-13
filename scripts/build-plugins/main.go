// Build the bundled plugin packages with standard Go and deterministic ZIP metadata.
package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gi8lino/lore/internal/pluginpackage"
)

func main() {
	if err := build(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func build() error {
	directories, err := filepath.Glob("plugins/*/plugin.yaml")
	if err != nil {
		return err
	}
	for _, manifest := range directories {
		if err := buildPackage(filepath.Dir(manifest)); err != nil {
			return err
		}
	}
	return nil
}

func buildPackage(directory string) error {
	temporary, err := os.MkdirTemp("", "lore-plugin-build-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(temporary) }()
	command := exec.Command("go", "build", "-trimpath", "-buildvcs=false", "-buildmode=c-shared", "-ldflags=-s -w -buildid=", "-o", filepath.Join(temporary, "plugin.wasm"), ".")
	command.Dir = directory
	// Reproduce the checked-in artifact with the manifest's Go toolchain version,
	// independently of the developer's newer patch version or ambient build flags.
	module, err := os.ReadFile(filepath.Join(directory, "go.mod"))
	if err != nil {
		return err
	}
	toolchain := ""
	for line := range strings.SplitSeq(string(module), "\n") {
		if value, ok := strings.CutPrefix(line, "go "); ok {
			toolchain = "go" + strings.TrimSpace(value)
		}
	}
	if toolchain == "" {
		return fmt.Errorf("no Go toolchain in %s", directory)
	}
	environment := make([]string, 0, len(os.Environ()))
	for _, value := range os.Environ() {
		key, _, _ := strings.Cut(value, "=")
		switch key {
		case "GOOS", "GOARCH", "GOFLAGS", "GOWORK", "GOTOOLCHAIN", "CGO_ENABLED":
			continue
		}
		environment = append(environment, value)
	}
	command.Env = append(environment, "GOOS=wasip1", "GOARCH=wasm", "CGO_ENABLED=0", "GOFLAGS=", "GOWORK=off", "GOTOOLCHAIN="+toolchain)
	if output, err := command.CombinedOutput(); err != nil {
		return fmt.Errorf("build %s: %w\n%s", directory, err, output)
	}
	files := map[string]string{"plugin.yaml": filepath.Join(directory, "plugin.yaml"), "plugin.wasm": filepath.Join(temporary, "plugin.wasm")}
	assets := filepath.Join(directory, "assets")
	if _, err := os.Stat(assets); err == nil {
		if err := filepath.WalkDir(assets, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				return nil
			}
			relative, err := filepath.Rel(directory, path)
			if err != nil {
				return err
			}
			files[filepath.ToSlash(relative)] = path
			return nil
		}); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	var buffer bytes.Buffer
	archive := zip.NewWriter(&buffer)
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		content, err := os.ReadFile(files[name])
		if err != nil {
			return err
		}
		header := &zip.FileHeader{Name: name, Method: zip.Deflate}
		header.SetMode(0644)
		writer, err := archive.CreateHeader(header)
		if err != nil {
			return err
		}
		if _, err := writer.Write(content); err != nil {
			return err
		}
	}
	if err := archive.Close(); err != nil {
		return err
	}
	if _, err := pluginpackage.Read(buffer.Bytes()); err != nil {
		return err
	}
	destination := filepath.Join("internal/plugins/bundled", filepath.Base(directory)+".loreplugin")
	previous, _ := os.ReadFile(destination)
	if bytes.Equal(previous, buffer.Bytes()) {
		return nil
	}
	return os.WriteFile(destination, buffer.Bytes(), 0644)
}
