//go:build !wasip1 || !wasm

package pluginapi

import "errors"

func Call(string, any, any) error { return errors.New("host capabilities require the WASM runtime") }
