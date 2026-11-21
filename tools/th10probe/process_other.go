//go:build !windows

package main

import "fmt"

func openProcess(uint32, string, bool) (process, error) {
	return nil, fmt.Errorf("process sampling is supported only on Windows; decoding tests are portable")
}
