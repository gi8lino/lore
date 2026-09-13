//go:build !wasip1 || !wasm

package pluginapi

import "errors"

// Call reports that Lore host capabilities are unavailable to native plugin code.
func Call(string, any, any) error { return errors.New("host capabilities require the WASM runtime") }
