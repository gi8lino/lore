// Package develop implements the lore-plugin development CLI.
package develop

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"runtime/debug"
	"strings"

	"github.com/gi8lino/lore/pluginapi"
	"github.com/gi8lino/lore/pluginsdk/build"
)

// Run runs a development command in the current working directory.
func Run(ctx context.Context, args []string, out, errOut io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: lore-plugin init <name> | test | build")
	}
	switch args[0] {
	case "init":
		return initialize(ctx, args[1:], out, errOut)
	case "test", "build":
		if len(args) != 1 {
			return fmt.Errorf("usage: lore-plugin %s", args[0])
		}
		directory, err := os.Getwd()
		if err != nil {
			return err
		}
		if _, err := build.Validate(directory); err != nil {
			return err
		}
		if args[0] == "test" {
			command := exec.CommandContext(ctx, "go", "test", "./...")
			command.Stdout = out
			command.Stderr = errOut
			if err := command.Run(); err != nil {
				return fmt.Errorf("go tests: %w", err)
			}
			return nil
		}
		destination := filepath.Join(directory, "dist", filepath.Base(directory)+".loreplugin")
		if err := build.Build(ctx, directory, destination); err != nil {
			return err
		}
		_, err = fmt.Fprintln(out, destination)
		return err
	default:
		return fmt.Errorf("unknown command %q; use init, test, or build", args[0])
	}
}

var projectName = regexp.MustCompile(`^[a-z][a-z0-9-]{0,62}$`)

func initialize(ctx context.Context, args []string, out, errOut io.Writer) error {
	flags := flag.NewFlagSet("init", flag.ContinueOnError)
	flags.SetOutput(errOut)
	sdkPath := flags.String("sdk-path", "", "local Lore checkout for SDK development")
	sdkVersion := flags.String("sdk-version", defaultVersion(), "Lore module version (defaults to this CLI's version)")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 1 || !projectName.MatchString(flags.Arg(0)) {
		return fmt.Errorf("usage: lore-plugin init [--sdk-version version | --sdk-path checkout] <lowercase-name>")
	}
	name := flags.Arg(0)
	version := *sdkVersion
	explicitVersion := false
	flags.Visit(func(value *flag.Flag) {
		if value.Name == "sdk-version" {
			explicitVersion = true
		}
	})
	if *sdkPath == "" && !explicitVersion {
		*sdkPath = developmentCheckout()
	}
	replacement := ""
	if *sdkPath != "" {
		absolute, err := filepath.Abs(*sdkPath)
		if err != nil {
			return err
		}
		if _, err := os.Stat(filepath.Join(absolute, "pluginsdk", "plugin.go")); err != nil {
			return fmt.Errorf("SDK checkout: %w", err)
		}
		replacement = fmt.Sprintf("\nreplace github.com/gi8lino/lore => %q\n", absolute)
		version = "v0.0.0"
	} else if version == "latest" {
		command := exec.CommandContext(ctx, "go", "list", "-m", "-f", "{{.Version}}", "github.com/gi8lino/lore@latest")
		value, err := command.Output()
		if err != nil {
			return fmt.Errorf("resolve SDK version (use --sdk-version or --sdk-path): %w", err)
		}
		version = strings.TrimSpace(string(value))
	}
	if !regexp.MustCompile(`^v[0-9][a-zA-Z0-9.+-]*$`).MatchString(version) {
		return fmt.Errorf("invalid SDK version %q", version)
	}
	if err := os.Mkdir(name, 0755); err != nil {
		return fmt.Errorf("create project: %w", err)
	}
	files := map[string]string{
		"go.mod":       fmt.Sprintf("module example.com/%s\n\ngo 1.27.0\n\nrequire github.com/gi8lino/lore %s\n%s", name, version, replacement),
		"plugin.yaml":  fmt.Sprintf("api_version: %d\nid: com.example.%s\nname: %s\nversion: 1.0.0\ndefault_enabled: true\nmodules:\n  - type: macro\n    id: greeting\n    name: greeting\npermissions: []\n", pluginapi.Version, name, name),
		"main.go":      sampleSource,
		"main_test.go": sampleTest,
		".gitignore":   "/dist/\n*.wasm\n",
		"README.md":    fmt.Sprintf("# %s\n\nA Go/WASI Lore plugin. Edit plugin.yaml to choose your unique plugin ID and metadata.\n\nUse `{{greeting}}` in a Lore page. Syntax in fenced code remains literal.\n\nRun `lore-plugin test`, then `lore-plugin build`. Install `dist/%s.loreplugin`\nfrom Administration → Plugins. No server rebuild or restart is required.\n\nRegister handlers in init; WASI reactor initialization does not run main.\nDeclare any capability permissions in plugin.yaml. Lore enforces grants and\nsanitizes all rendered output. Compile browser assets into assets/ before building.\n", name, name),
	}
	for path, content := range files {
		if err := os.WriteFile(filepath.Join(name, path), []byte(content), 0644); err != nil {
			return err
		}
	}
	command := exec.CommandContext(ctx, "go", "mod", "tidy")
	command.Dir = name
	command.Stdout = out
	command.Stderr = errOut
	if err := command.Run(); err != nil {
		return fmt.Errorf("resolve SDK in %s: %w (project retained; correct the dependency and run go mod tidy)", name, err)
	}
	_, err := fmt.Fprintf(out, "Created %s. Run: cd %s && lore-plugin test && lore-plugin build\n", name, name)
	return err
}

func defaultVersion() string {
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return "latest"
}

const sampleSource = `package main

import (
 "strings"
 plugin "github.com/gi8lino/lore/pluginsdk"
)

func main() {}

func init() {
 plugin.RegisterMacro("greeting", parseGreeting, renderGreeting)
}

func parseGreeting(source string) (struct{}, bool) {
 return struct{}{}, strings.TrimSpace(source) == "{{greeting}}"
}

func renderGreeting(_ struct{}) (plugin.Result, error) {
 return plugin.Text("<p>Hello from a Go plugin!</p>"), nil
}
`

const sampleTest = `package main

import "testing"

func TestGreeting(t *testing.T) {
 if _, matched := parseGreeting("{{greeting}}"); !matched { t.Fatal("greeting should match") }
 if _, matched := parseGreeting("ordinary text"); matched { t.Fatal("ordinary text should not match") }
 result, err := renderGreeting(struct{}{})
 if err != nil || len(result.Parts) != 1 { t.Fatalf("render: %+v, %v", result, err) }
}
`

// developmentCheckout keeps an unreleased checkout-built CLI usable without
// resolving an older published Lore version that may not contain this SDK.
func developmentCheckout() string {
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		return ""
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(source), "../.."))
	if _, err := os.Stat(filepath.Join(root, ".git")); err != nil {
		return ""
	}
	module, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil || !strings.HasPrefix(string(module), "module github.com/gi8lino/lore\n") {
		return ""
	}
	if _, err := os.Stat(filepath.Join(root, "pluginsdk", "plugin.go")); err != nil {
		return ""
	}
	return root
}
