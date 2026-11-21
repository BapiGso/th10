//go:build js

package browserstore

import (
	"bytes"
	"errors"
	"io/fs"
	"strings"
	"syscall/js"
	"testing"
	"time"
)

func indexedDBTestControl(t *testing.T) js.Value {
	t.Helper()
	control := js.Global().Get("th10IndexedDBTest")
	if !control.Truthy() {
		t.Skip("requires tools/wasmexec's explicitly enabled fake IndexedDB")
	}
	control.Call("reset")
	return control
}

func TestIndexedDBReadWriteReloadAndCopy(t *testing.T) {
	indexedDBTestControl(t)
	store, err := Open()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Read("score.dat"); !errors.Is(err, fs.ErrNotExist) {
		t.Fatal("missing key did not report not-exist")
	}
	data := []byte{1, 2, 3}
	if err := store.Write("score.dat", data); err != nil {
		t.Fatal(err)
	}
	data[0] = 9
	again, err := Open()
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := again.Read("score.dat")
	if err != nil || !bytes.Equal(loaded, []byte{1, 2, 3}) {
		t.Fatal("second connection did not reload committed bytes")
	}
	loaded[0] = 8
	loaded, err = store.Read("score.dat")
	if err != nil || loaded[0] != 1 {
		t.Fatal("returned bytes aliased the database")
	}
}

func TestIndexedDBWriteWaitsForTransactionCommit(t *testing.T) {
	control := indexedDBTestControl(t)
	store, err := Open()
	if err != nil {
		t.Fatal(err)
	}
	control.Call("holdWrites", true)
	defer control.Call("flushWrites")
	done := make(chan error, 1)
	go func() { done <- store.Write("score.dat", []byte{4, 5, 6}) }()
	deadline := time.Now().Add(2 * time.Second)
	for control.Call("pendingWrites").Int() == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if control.Call("pendingWrites").Int() != 1 {
		t.Fatal("put request did not reach the held transaction")
	}
	select {
	case err := <-done:
		t.Fatalf("Write returned on request success instead of transaction completion: %v", err)
	default:
	}
	control.Call("flushWrites")
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Write did not finish after commit")
	}
}

func TestIndexedDBAbortKeepsExistingBytes(t *testing.T) {
	control := indexedDBTestControl(t)
	store, err := Open()
	if err != nil {
		t.Fatal(err)
	}
	old := []byte{7, 8, 9}
	if err := store.Write("replay/th10_01.rpy", old); err != nil {
		t.Fatal(err)
	}
	control.Call("failNextCommit")
	if err := store.Write("replay/th10_01.rpy", []byte{0}); err == nil || !strings.Contains(err.Error(), "QuotaExceededError") {
		t.Fatalf("commit failure was acknowledged or lost its reason: %v", err)
	}
	loaded, err := store.Read("replay/th10_01.rpy")
	if err != nil || !bytes.Equal(loaded, old) {
		t.Fatal("aborted transaction replaced the old replay")
	}
	if err := store.Write("too-big", make([]byte, MaxBlobBytes+1)); err == nil {
		t.Fatal("oversized data reached IndexedDB")
	}
}
