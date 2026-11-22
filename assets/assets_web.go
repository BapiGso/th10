//go:build js && th10web

package assets

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"sync"
	"syscall/js"
	"time"
)

// TextureVariant identifies the selected pixel policy for resource tests.
const TextureVariant = "web"

// prefetchWindow is how many background pack fetches may be in flight.
const prefetchWindow = 3

var store = struct {
	sync.Mutex
	manifest Manifest
	started  bool
	packs    map[string]func() error // pack name -> its one load
	files    map[string][]byte
	textures map[string][]byte
	loaded   int
	total    int // bytes of every pack in the manifest
	count    int
	report   func(Progress)
}{packs: map[string]func() error{}, files: map[string][]byte{}, textures: map[string][]byte{}}

// Start fetches the manifest and the boot pack, then prefetches the other
// packs in PackOrder in the background, a few at a time. It must run before the game reads
// any resource and must not run inside a js.FuncOf callback.
func Start(report func(Progress)) error {
	store.Lock()
	if store.started {
		store.Unlock()
		return nil
	}
	store.started = true
	if report == nil {
		report = func(Progress) {}
	}
	store.report = report
	store.Unlock()
	data, err := fetchBytes(ManifestPath, "no-cache", "")
	if err != nil {
		return fmt.Errorf("asset manifest: %w", err)
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return fmt.Errorf("asset manifest: %w", err)
	}
	if _, ok := m.Packs[BootPack]; !ok {
		return errors.New("asset manifest has no boot pack")
	}
	total := 0
	for _, p := range m.Packs {
		total += p.Bytes
	}
	store.Lock()
	store.manifest, store.total = m, total
	store.Unlock()
	if err := ensure(BootPack); err != nil {
		return err
	}
	// Fetches are native and off the Go thread; keeping a few in flight hides
	// the per-request latency without losing the play order.
	queue := make(chan string)
	for range prefetchWindow {
		go func() {
			for name := range queue {
				_ = ensure(name) // a failure is reported; a later read retries nothing and returns it
			}
		}()
	}
	go func() {
		for _, name := range PackOrder() {
			queue <- name
		}
		for name := range m.Packs { // packs added to the manifest but not to PackOrder
			queue <- name
		}
		close(queue)
	}()
	return nil
}

// ensure blocks until pack name is loaded, fetching it now if neither the
// prefetcher nor another reader has started it.
func ensure(name string) error {
	store.Lock()
	load, ok := store.packs[name]
	if !ok {
		load = sync.OnceValue(func() error { return loadPack(name) })
		store.packs[name] = load
	}
	store.Unlock()
	return load()
}

func loadPack(name string) error {
	store.Lock()
	info, ok := store.manifest.Packs[name]
	report := store.report
	store.Unlock()
	if !ok {
		return nil // nothing published under this name (e.g. no common files)
	}
	// The browser checks the SHA-256 (Subresource Integrity) natively, off
	// the single Go thread; hashing in Go cost ~17 ms of frame time per 2 MB.
	sum, err := hex.DecodeString(info.SHA256)
	integrity := "sha256-" + base64.StdEncoding.EncodeToString(sum)
	var data []byte
	for attempt := range 3 {
		if err != nil {
			break
		}
		if attempt > 0 {
			time.Sleep(time.Duration(attempt) * time.Second)
		}
		if data, err = fetchBytes("assets/"+info.File, "default", integrity); err == nil && len(data) != info.Bytes {
			err = fmt.Errorf("size differs from the manifest")
		}
		if err == nil {
			break
		}
	}
	var entries []PackEntry
	if err == nil {
		entries, err = DecodePack(data)
	}
	if err != nil {
		err = fmt.Errorf("asset pack %s: %w", name, err)
		report(Progress{Pack: name, Err: err})
		return err
	}
	store.Lock()
	for _, e := range entries {
		if e.Kind == PackTexture {
			store.textures[e.Name] = e.Data
		} else {
			store.files[e.Name] = e.Data
		}
	}
	store.loaded += info.Bytes
	store.count++
	progress := Progress{Pack: name, Bytes: info.Bytes, LoadedBytes: store.loaded, TotalBytes: store.total, LoadedPacks: store.count, TotalPacks: len(store.manifest.Packs)}
	store.Unlock()
	report(progress)
	return nil
}

// ReadFile returns a private copy of a runtime file, waiting for its pack.
func ReadFile(name string) ([]byte, error) {
	pack, start := PackOf(name), time.Now()
	err := ensure(pack)
	if waited := time.Since(start); waited > 20*time.Millisecond {
		// The game thread stalled on the network: the prefetch order missed this read.
		js.Global().Get("console").Call("info", fmt.Sprintf("[TH10] %s waited %.0f ms for asset pack %s", name, waited.Seconds()*1000, pack))
	}
	if err != nil {
		return nil, &fs.PathError{Op: "open", Path: name, Err: err}
	}
	store.Lock()
	data, ok := store.files[name]
	store.Unlock()
	if !ok {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}
	return bytes.Clone(data), nil
}

// ReadTexture returns a texture blob. ANMs are restored only after their own
// pack, which carries every texture they reference, has loaded.
func ReadTexture(id string) ([]byte, error) {
	store.Lock()
	data, ok := store.textures[id]
	store.Unlock()
	if !ok {
		return nil, fmt.Errorf("assets: texture %s not in a loaded pack", id)
	}
	return data, nil
}

// fetchBytes waits for fetch(url) without blocking the JS event loop: the
// promise callbacks only hand results to this goroutine.
func fetchBytes(url, cache, integrity string) ([]byte, error) {
	type result struct {
		value js.Value
		err   error
	}
	await := func(promise js.Value) (js.Value, error) {
		ch := make(chan result, 1)
		ok := js.FuncOf(func(_ js.Value, args []js.Value) any {
			ch <- result{value: args[0]}
			return nil
		})
		fail := js.FuncOf(func(_ js.Value, args []js.Value) any {
			ch <- result{err: errors.New(js.Global().Get("String").Invoke(args[0]).String())}
			return nil
		})
		promise.Call("then", ok, fail)
		r := <-ch
		ok.Release()
		fail.Release()
		return r.value, r.err
	}
	options := js.Global().Get("Object").New()
	options.Set("cache", cache)
	if integrity != "" {
		options.Set("integrity", integrity)
	}
	response, err := await(js.Global().Call("fetch", url, options))
	if err != nil {
		return nil, err
	}
	if !response.Get("ok").Bool() {
		return nil, fmt.Errorf("HTTP %d for %s", response.Get("status").Int(), url)
	}
	buffer, err := await(response.Call("arrayBuffer"))
	if err != nil {
		return nil, err
	}
	array := js.Global().Get("Uint8Array").New(buffer)
	data := make([]byte, array.Get("byteLength").Int())
	js.CopyBytesToGo(data, array)
	return data, nil
}
