//go:build windows

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

type windowsProcess struct {
	h   windows.Handle
	pid uint32
}

func (p *windowsProcess) ID() uint32   { return p.pid }
func (p *windowsProcess) Close() error { return windows.CloseHandle(p.h) }
func (p *windowsProcess) ReadMemory(addr uint32, dst []byte) error {
	if len(dst) == 0 {
		return nil
	}
	var n uintptr
	if err := windows.ReadProcessMemory(p.h, uintptr(addr), &dst[0], uintptr(len(dst)), &n); err != nil {
		return err
	}
	if int(n) != len(dst) {
		return io.ErrUnexpectedEOF
	}
	return nil
}

func openProcess(pid uint32, exePath string, skipHash bool) (process, error) {
	if pid == 0 {
		if exePath == "" {
			return nil, fmt.Errorf("need -pid or -exe")
		}
		var err error
		pid, err = findPIDByExe(exePath)
		if err != nil {
			return nil, err
		}
	}
	actual, err := modulePathForPID(pid)
	if err != nil {
		return nil, err
	}
	if exePath != "" {
		want, err := filepath.Abs(exePath)
		if err != nil {
			return nil, err
		}
		if !strings.EqualFold(filepath.Clean(want), filepath.Clean(actual)) {
			return nil, fmt.Errorf("pid %d runs %q, not %q", pid, actual, want)
		}
	}
	if !skipHash {
		if err := verifyHash(actual); err != nil {
			return nil, err
		}
	}
	base, err := moduleBase(pid, actual)
	if err != nil {
		return nil, err
	}
	if base != imageBase {
		return nil, fmt.Errorf("module base 0x%x != expected 0x%x", base, imageBase)
	}
	// No write/operation/thread rights are requested, even for shutdown.
	h, err := windows.OpenProcess(windows.PROCESS_VM_READ|windows.PROCESS_QUERY_INFORMATION, false, pid)
	if err != nil {
		return nil, fmt.Errorf("OpenProcess (read/query): %w", err)
	}
	return &windowsProcess{h: h, pid: pid}, nil
}

func findPIDByExe(exePath string) (uint32, error) {
	want, err := filepath.Abs(exePath)
	if err != nil {
		return 0, err
	}
	snap, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return 0, err
	}
	defer func() { _ = windows.CloseHandle(snap) }()
	var e windows.ProcessEntry32
	e.Size = uint32(unsafe.Sizeof(e))
	if err := windows.Process32First(snap, &e); err != nil {
		return 0, err
	}
	for {
		if strings.EqualFold(windows.UTF16ToString(e.ExeFile[:]), filepath.Base(want)) {
			path, err := modulePathForPID(e.ProcessID)
			if err == nil && strings.EqualFold(filepath.Clean(path), filepath.Clean(want)) {
				return e.ProcessID, nil
			}
		}
		if windows.Process32Next(snap, &e) != nil {
			break
		}
	}
	return 0, fmt.Errorf("process %q not running", want)
}

func modulePathForPID(pid uint32) (string, error) {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return "", fmt.Errorf("query process path: %w", err)
	}
	defer func() { _ = windows.CloseHandle(h) }()
	var buf [32768]uint16
	n := uint32(len(buf))
	if err := windows.QueryFullProcessImageName(h, 0, &buf[0], &n); err != nil {
		return "", err
	}
	return windows.UTF16ToString(buf[:n]), nil
}

func moduleBase(pid uint32, exePath string) (uintptr, error) {
	snap, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPMODULE|windows.TH32CS_SNAPMODULE32, pid)
	if err != nil {
		return 0, fmt.Errorf("query module base: %w", err)
	}
	defer func() { _ = windows.CloseHandle(snap) }()
	var m windows.ModuleEntry32
	m.Size = uint32(unsafe.Sizeof(m))
	if err := windows.Module32First(snap, &m); err != nil {
		return 0, err
	}
	for {
		if strings.EqualFold(windows.UTF16ToString(m.ExePath[:]), exePath) {
			return uintptr(m.ModBaseAddr), nil
		}
		if windows.Module32Next(snap, &m) != nil {
			break
		}
	}
	return 0, fmt.Errorf("module %q not found in pid %d", exePath, pid)
}

func verifyHash(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	got := hex.EncodeToString(h.Sum(nil))
	if got != ExpectedHash {
		return fmt.Errorf("th10.exe SHA-256 mismatch: got %s, want %s", got, ExpectedHash)
	}
	return nil
}
