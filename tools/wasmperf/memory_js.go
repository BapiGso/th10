//go:build js

package main

import "syscall/js"

func wasmMemoryBytes() uint64 {
	if read := js.Global().Get("th10WASMMemoryBytes"); read.Type() == js.TypeFunction {
		return uint64(read.Invoke().Float())
	}
	return 0
}
