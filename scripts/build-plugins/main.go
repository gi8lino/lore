// Build the bundled plugin packages with standard Go and deterministic ZIP metadata.
package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/gi8lino/lore/pluginsdk/build"
)

// main runs the package entry point.
func main() {
	if err := buildAll(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// buildAll discovers bundled plugin sources and rebuilds their deterministic archives.
func buildAll() error {
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

// buildPackage writes one deterministic bundled plugin archive.
func buildPackage(directory string) error {
	if _, err := os.Stat(filepath.Join(directory, "browser.ts")); err == nil {
		command := exec.Command(
			"./node_modules/.bin/tsc",
			"--ignoreConfig",
			filepath.Join(directory, "browser.ts"),
			"--target",
			"ES2022",
			"--lib",
			"ES2022,DOM,DOM.Iterable",
			"--strict",
			"--outDir",
			filepath.Join(directory, "assets"),
		)
		if output, err := command.CombinedOutput(); err != nil {
			return fmt.Errorf("build plugin browser entry: %w: %s", err, output)
		}
		if err := os.Rename(filepath.Join(directory, "assets", "browser.js"), filepath.Join(directory, "assets", "plugin.js")); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}

	return build.Build(context.Background(), directory, filepath.Join("internal/firstparty", filepath.Base(directory)+".loreplugin"))
}
