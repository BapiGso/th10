//go:build js

package browserstore

import (
	"errors"
	"fmt"
	"io/fs"
	"sync/atomic"
	"syscall/js"
)

const (
	databaseName = "th10"
	objectStore  = "files"
)

type indexedStore struct {
	db      js.Value
	closed  atomic.Bool
	version js.Func
}

type eventResult struct {
	value js.Value
	err   error
}

func Open() (store Store, err error) {
	defer catchJS("open", &err)
	indexedDB := js.Global().Get("indexedDB")
	if !indexedDB.Truthy() {
		return nil, ErrUnavailable
	}
	request := indexedDB.Call("open", databaseName, 1)
	done := make(chan eventResult, 1)
	var upgradeErr error
	success := js.FuncOf(func(js.Value, []js.Value) any {
		done <- eventResult{value: request.Get("result"), err: upgradeErr}
		return nil
	})
	failure := js.FuncOf(func(js.Value, []js.Value) any {
		done <- eventResult{err: errorValue(request.Get("error"))}
		return nil
	})
	upgrade := js.FuncOf(func(js.Value, []js.Value) any {
		func() {
			defer func() {
				if value := recover(); value != nil {
					upgradeErr = fmt.Errorf("browserstore: upgrade: %v", value)
					request.Get("transaction").Call("abort")
				}
			}()
			db := request.Get("result")
			if !db.Get("objectStoreNames").Call("contains", objectStore).Bool() {
				db.Call("createObjectStore", objectStore)
			}
		}()
		return nil
	})
	blocked := js.FuncOf(func(js.Value, []js.Value) any {
		notify("open", "", 0, errors.New("database upgrade is blocked; close other TH10 tabs"))
		return nil
	})
	request.Set("onsuccess", success)
	request.Set("onerror", failure)
	request.Set("onupgradeneeded", upgrade)
	request.Set("onblocked", blocked)
	result := <-done
	for _, name := range []string{"onsuccess", "onerror", "onupgradeneeded", "onblocked"} {
		request.Set(name, js.Null())
	}
	success.Release()
	failure.Release()
	upgrade.Release()
	blocked.Release()
	if result.err != nil {
		notify("open", "", 0, result.err)
		return nil, result.err
	}
	s := &indexedStore{db: result.value}
	s.version = js.FuncOf(func(js.Value, []js.Value) any {
		s.closed.Store(true)
		s.db.Call("close")
		notify("close", "", 0, errors.New("database version changed; reload this page"))
		return nil
	})
	s.db.Set("onversionchange", s.version)
	notify("open", "", 0, nil)
	return s, nil
}

func (s *indexedStore) Read(key string) (data []byte, err error) {
	defer catchJS("read "+key, &err)
	if s.closed.Load() {
		return nil, errors.New("browserstore: database is closed; reload this page")
	}
	tx := s.db.Call("transaction", objectStore, "readonly")
	request := tx.Call("objectStore", objectStore).Call("get", key)
	value, err := awaitTransaction(tx, request)
	if err != nil {
		notify("read", key, 0, err)
		return nil, err
	}
	if value.IsUndefined() {
		return nil, fs.ErrNotExist
	}
	if value.Type() != js.TypeObject || !value.InstanceOf(js.Global().Get("ArrayBuffer")) {
		return nil, fmt.Errorf("browserstore: %s is not binary data", key)
	}
	length := value.Get("byteLength").Int()
	if length < 0 || length > MaxBlobBytes {
		return nil, fmt.Errorf("browserstore: %s exceeds %d bytes", key, MaxBlobBytes)
	}
	data = make([]byte, length)
	js.CopyBytesToGo(data, js.Global().Get("Uint8Array").New(value))
	return data, nil
}

func (s *indexedStore) Write(key string, data []byte) (err error) {
	defer func() {
		if err != nil {
			notify("write", key, len(data), err)
		}
	}()
	defer catchJS("write "+key, &err)
	if s.closed.Load() {
		return errors.New("browserstore: database is closed; reload this page")
	}
	if len(data) > MaxBlobBytes {
		return fmt.Errorf("browserstore: %s exceeds %d bytes", key, MaxBlobBytes)
	}
	array := js.Global().Get("Uint8Array").New(len(data))
	js.CopyBytesToJS(array, data)
	tx := s.db.Call("transaction", objectStore, "readwrite")
	request := tx.Call("objectStore", objectStore).Call("put", array.Get("buffer"), key)
	if _, err = awaitTransaction(tx, request); err != nil {
		return err
	}
	notify("write", key, len(data), nil)
	return nil
}

func awaitTransaction(tx, request js.Value) (js.Value, error) {
	done := make(chan eventResult, 1)
	var requestErr error
	complete := js.FuncOf(func(js.Value, []js.Value) any {
		done <- eventResult{value: request.Get("result")}
		return nil
	})
	abort := js.FuncOf(func(js.Value, []js.Value) any {
		err := requestErr
		if err == nil {
			err = errorValue(tx.Get("error"))
		}
		done <- eventResult{err: err}
		return nil
	})
	failure := js.FuncOf(func(js.Value, []js.Value) any {
		problem := request.Get("error")
		if problem.IsNull() || problem.IsUndefined() {
			problem = tx.Get("error")
		}
		requestErr = errorValue(problem)
		return nil // Do not preventDefault: the transaction must abort, not partially commit.
	})
	tx.Set("oncomplete", complete)
	tx.Set("onabort", abort)
	tx.Set("onerror", failure)
	result := <-done
	tx.Set("oncomplete", js.Null())
	tx.Set("onabort", js.Null())
	tx.Set("onerror", js.Null())
	complete.Release()
	abort.Release()
	failure.Release()
	return result.value, result.err
}

func errorValue(value js.Value) error {
	if value.Type() == js.TypeObject && !value.IsNull() {
		return fmt.Errorf("browserstore: %s: %s", value.Get("name").String(), value.Get("message").String())
	}
	return errors.New("browserstore: IndexedDB transaction failed")
}

func catchJS(operation string, err *error) {
	if value := recover(); value != nil {
		if exception, ok := value.(js.Error); ok {
			*err = fmt.Errorf("browserstore: %s: %s", operation, exception.Error())
			return
		}
		panic(value)
	}
}

func notify(operation, key string, size int, err error) {
	message := map[string]any{"operation": operation, "key": key, "bytes": size, "error": ""}
	method, text := "info", "committed"
	if operation == "open" {
		text = "ready"
	}
	if err != nil {
		method, text = "error", err.Error()
		message["error"] = text
	}
	js.Global().Get("console").Call(method, "[TH10 storage]", operation, key, size, text)
	if listener := js.Global().Get("th10StorageEvent"); listener.Type() == js.TypeFunction {
		listener.Invoke(message)
	}
}
