//go:build wasip1 && wasm

package main

import (
	"encoding/json"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"
	"unsafe"

	"github.com/gi8lino/lore/pluginapi"
)

var input, output []byte

func main() {}

//go:wasmexport lore_api_version
func version() uint32 { return 1 }

//go:wasmexport lore_alloc
func alloc(n uint32) uint32 {
	input = make([]byte, n)
	return uint32(uintptr(unsafe.Pointer(&input[0])))
}

//go:wasmexport lore_transform
func transform(pointer, length uint32) uint64 {
	var request pluginapi.RenderRequest
	_ = json.Unmarshal(input, &request)
	result := pluginapi.RenderResult{Parts: []pluginapi.RenderPart{{Text: request.Source}}}
	switch {
	case request.Source == "loop":
		for {
		}
	case request.Source == "trap":
		panic("guest failed")
	case request.Source == "bad-pointer":
		return uint64(100)<<32 | 0xffffffff
	case request.Source == "oversized":
		return uint64(0xffffffff) << 32
	case request.Source == "malformed":
		output = []byte("{")
		return address()
	case request.Source == "trailing":
		output = []byte(`{} {}`)
		return address()
	case request.Source == "grow":
		output = make([]byte, 128<<20)
		return address()
	case request.Source == "unknown-field":
		output = []byte(`{"trusted_html":"<script>bad</script>"}`)
		return address()
	case request.Source == "recursive":
		value := "recursive"
		result.Parts = []pluginapi.RenderPart{{Markdown: &value}}
	case request.Source == "unsafe":
		result.Parts = []pluginapi.RenderPart{{Text: `<div>safe</div><script>bad()</script><a href="javascript:bad()">link</a>`}}
	case request.Source == "environment":
		result.Parts[0].Text = strings.Join(os.Environ(), ",")
	case strings.HasPrefix(request.Source, "read:"):
		_, err := os.ReadFile(strings.TrimPrefix(request.Source, "read:"))
		result.Parts[0].Text = denied(err)
	case strings.HasPrefix(request.Source, "write:"):
		err := os.WriteFile(strings.TrimPrefix(request.Source, "write:"), []byte("modified"), 0600)
		result.Parts[0].Text = denied(err)
	case strings.HasPrefix(request.Source, "network:"):
		connection, err := net.DialTimeout("tcp", strings.TrimPrefix(request.Source, "network:"), 20*time.Millisecond)
		if connection != nil {
			_ = connection.Close()
		}
		result.Parts[0].Text = denied(err)
	case request.Source == "process":
		result.Parts[0].Text = denied(exec.Command("/bin/sh", "-c", "exit 0").Run())
	}
	output, _ = json.Marshal(result)
	return address()
}
func denied(err error) string {
	if err != nil {
		return "denied"
	}
	return "ALLOWED"
}
func address() uint64 { return uint64(len(output))<<32 | uint64(uintptr(unsafe.Pointer(&output[0]))) }
