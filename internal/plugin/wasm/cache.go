package wasm

import (
	"context"
	"crypto/sha256"
	"sync"

	"github.com/tetratelabs/wazero"
)

// Cache immutable decoded/compiled modules, not just machine code. Repeated
// CompileModule calls otherwise decode multi-megabyte Go binaries each time.
// Leases keep evicted modules alive until their last instance closes. All
// instances still own separate linear memory and request capabilities.
var retainedCode []*codeEntry
var codeMu sync.Mutex

type codeEntry struct {
	digest  [32]byte
	pages   uint32
	module  wazero.CompiledModule
	refs    int
	retired bool
}
type compiledLease struct {
	wazero.CompiledModule
	entry *codeEntry
	once  sync.Once
}

func (l *compiledLease) Close(ctx context.Context) error {
	var err error
	l.once.Do(func() {
		codeMu.Lock()
		defer codeMu.Unlock()
		l.entry.refs--
		if l.entry.retired && l.entry.refs == 0 {
			err = l.entry.module.Close(ctx)
		}
	})
	return err
}

// compile is called with compilationGate held. Wazero permits a compiled module
// to instantiate in another runtime sharing the same compilation configuration.
func (r *Runtime) compile(ctx context.Context, binary []byte) (*compiledLease, error) {
	digest := sha256.Sum256(binary)
	codeMu.Lock()
	for index, entry := range retainedCode {
		if entry.digest == digest && entry.pages == r.limits.MemoryPages {
			copy(retainedCode[index:], retainedCode[index+1:])
			retainedCode[len(retainedCode)-1] = entry
			entry.refs++
			codeMu.Unlock()
			return &compiledLease{CompiledModule: entry.module, entry: entry}, nil
		}
	}
	codeMu.Unlock()
	module, err := r.engine.CompileModule(ctx, binary)
	if err != nil {
		return nil, err
	}
	if err := validateABI(module); err != nil {
		_ = module.Close(ctx)
		return nil, err
	}
	codeMu.Lock()
	defer codeMu.Unlock()
	if len(retainedCode) == 8 {
		entry := retainedCode[0]
		entry.retired = true
		if entry.refs == 0 {
			_ = entry.module.Close(context.Background())
		}
		retainedCode = retainedCode[1:]
	}
	entry := &codeEntry{digest: digest, pages: r.limits.MemoryPages, module: module, refs: 1}
	retainedCode = append(retainedCode, entry)
	return &compiledLease{CompiledModule: module, entry: entry}, nil
}
