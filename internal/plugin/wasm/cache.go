package wasm

import (
	"context"
	"crypto/sha256"

	"github.com/tetratelabs/wazero"
)

// Keep a bounded number of compiled-code references alive between runtime
// scopes. Wazero evicts code when its last CompiledModule closes, so its cache
// alone cannot reuse code across successive short-lived renderers. Entries own
// code only, never instantiated memory, registries, or request capabilities.
// Access is serialized by compilationGate.
var retainedCode []codeEntry

type codeEntry struct {
	digest [32]byte
	pages  uint32
	module wazero.CompiledModule
}

func (r *Runtime) retainCode(ctx context.Context, binary []byte) error {
	digest := sha256.Sum256(binary)
	for index, entry := range retainedCode {
		if entry.digest == digest && entry.pages == r.limits.MemoryPages {
			copy(retainedCode[index:], retainedCode[index+1:])
			retainedCode[len(retainedCode)-1] = entry
			return nil
		}
	}
	// A second compile is a cache hit and owns an independent reference. Closing
	// the plugin's own compiled module therefore cannot evict the cached code.
	reference, err := r.engine.CompileModule(ctx, binary)
	if err != nil {
		return err
	}
	const capacity = 8
	if len(retainedCode) == capacity {
		_ = retainedCode[0].module.Close(context.Background())
		retainedCode = retainedCode[1:]
	}
	retainedCode = append(retainedCode, codeEntry{digest, r.limits.MemoryPages, reference})
	return nil
}
