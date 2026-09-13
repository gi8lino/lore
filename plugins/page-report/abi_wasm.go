//go:build wasip1 && wasm

package main

import (
	"encoding/json"
	"unsafe"

	"github.com/gi8lino/lore/pluginapi"
)

// The host serializes calls. These slices keep the ABI buffers alive until the
// next invocation. No host pointers or Go values cross the WASM boundary.
var input, output []byte

//go:wasmexport lore_api_version
func apiVersion() uint32 { return pluginapi.Version }

//go:wasmexport lore_alloc
func allocate(size uint32) uint32 {
	if size == 0 || size > 4<<20 {
		return 0
	}
	input = make([]byte, size)
	return uint32(uintptr(unsafe.Pointer(&input[0])))
}

//go:wasmexport lore_transform
func invoke(pointer, length uint32) uint64 {
	var result pluginapi.RenderResult
	if len(input) == 0 || pointer != uint32(uintptr(unsafe.Pointer(&input[0]))) || uint64(length) != uint64(len(input)) {
		result.Error = "invalid request buffer"
	} else {
		var request pluginapi.RenderRequest
		if err := json.Unmarshal(input, &request); err != nil {
			result.Error = "invalid request JSON"
		} else {
			result = transform(request)
		}
	}
	output, _ = json.Marshal(result)
	return uint64(len(output))<<32 | uint64(uintptr(unsafe.Pointer(&output[0])))
}
