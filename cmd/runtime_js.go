//go:build js

package main

import (
	"fmt"
	"io"
	goruntime "runtime"
	"syscall/js"
	thaudio "th10/audio"
	"th10/game"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

func preparePlatform() error {
	if err := game.InitBrowserStorage(); err != nil {
		platformMessage("th10Fatal", err.Error())
		return err
	}
	return nil
}

func runPlatformGame(g *game.Game) error {
	// Gate-time Draws keep the clock at zero. A single SyncWithFPS bootstrap
	// tick starts exactly one real frame, then fixed 60 TPS takes over.
	ebiten.SetRunnableOnUnfocused(true)
	g.Audio().SetBGMBufferSize(200 * time.Millisecond)
	runtime := &gatedGame{inner: g, audioReady: g.AudioContext().IsReady, waitForDraw: true}
	runtime.resumeClock = func() { ebiten.SetTPS(ebiten.SyncWithFPS) }
	runtime.onReady = func() {
		ebiten.SetTPS(0)
		// Warm the decoder/JIT before any sig/title frame. No player or music
		// observer is touched, and only 200 ms of PCM is read, not an entire song.
		stream, err := thaudio.NewBGMStream(thaudio.BGMTitle, game.SampleRate)
		if err == nil {
			_, err = io.ReadFull(stream, make([]byte, game.SampleRate*4/5))
		}
		if err != nil {
			platformMessage("th10Fatal", "Vorbis prewarm: "+err.Error())
		} else {
			js.Global().Get("console").Call("info", "[TH10] Vorbis warmed before first game frame; BGM source buffer=200ms")
		}
		platformMessage("th10RuntimeReady", map[string]any{"scoreLimit": game.MaxScoreImportBytes, "replayLimit": game.MaxReplayImportBytes})
		if g.Save.LoadError != "" {
			platformMessage("th10Fatal", "score load: "+g.Save.LoadError)
		}
	}
	runtime.onStart = func() {
		ebiten.SetTPS(60)
		ebiten.SetRunnableOnUnfocused(false)
		g.StartBrowserClock()
		js.Global().Get("console").Call("info", "[TH10] first game frame; audio.IsReady=true; TPS=60; unfocused updates=false")
		platformMessage("th10Started", true)
	}
	runtime.onMetrics = func(m runtimeMetrics) {
		var memory goruntime.MemStats
		goruntime.ReadMemStats(&memory)
		runtime.stats.setHeap(memory.HeapAlloc)
		m.HeapBytes = memory.HeapAlloc
		js.Global().Get("console").Call("info", fmt.Sprintf("[TH10 runtime] TPS=%.2f FPS=%.2f Update last/mean/max=%.3f/%.3f/%.3fms Draw=%.3f/%.3f/%.3fms GoHeap=%d", m.TPS, m.FPS, m.UpdateMS, m.UpdateMeanMS, m.UpdateMaxMS, m.DrawMS, m.DrawMeanMS, m.DrawMaxMS, m.HeapBytes))
		platformMessage("th10Metrics", metricsObject(m))
	}
	api := newBrowserAPI(g, runtime)
	defer api.close()
	if err := ebiten.RunGame(runtime); err != nil {
		platformMessage("th10Fatal", err.Error())
		return err
	}
	// FUN_00438ad0's normal cfg exit write and FUN_0041fb50's score write.
	// Page close/refresh is not a reliable asynchronous transaction checkpoint.
	if err := g.SaveConfig(); err != nil {
		return err
	}
	return g.SaveProgress()
}

type browserAPI struct{ functions []js.Func }

func newBrowserAPI(g *game.Game, runtime *gatedGame) *browserAPI {
	api := &browserAPI{}
	object := js.Global().Get("Object").New()
	api.bind(object, "start", func(_ js.Value, _ []js.Value) any {
		if err := runtime.gate.request(); err != nil {
			return err.Error()
		}
		return ""
	})
	api.bind(object, "toggleDebug", func(_ js.Value, _ []js.Value) any { return runtime.toggleDebug() })
	api.bind(object, "diagnostics", func(_ js.Value, _ []js.Value) any {
		m := metricsObject(runtime.stats.view())
		m["started"], m["audioReady"], m["configuredTPS"], m["focused"] = runtime.gate.isStarted(), g.AudioContext().IsReady(), ebiten.TPS(), ebiten.IsFocused()
		m["bgmBufferMs"] = float64(g.Audio().BGMBufferSize()) / float64(time.Millisecond)
		if memory := js.Global().Get("th10WASMMemoryBytes"); memory.Type() == js.TypeFunction {
			m["linearMemoryBytes"] = memory.Invoke().Float()
		}
		return m
	})
	api.bind(object, "importFile", func(_ js.Value, args []js.Value) any {
		if err := runtime.gate.beginImport(); err != nil {
			return browserPromise(func() (any, error) { return nil, err })
		}
		return browserPromise(func() (any, error) {
			defer runtime.gate.endImport()
			kind, slot, data, err := importArguments(args)
			if err != nil {
				return nil, err
			}
			switch kind {
			case "score":
				if err := g.ImportScore(data); err != nil {
					return nil, err
				}
				g.LoadProgress() // No scene has run yet, so no active state keeps the old history.
			case "replay":
				if err := g.ImportReplay(slot, data); err != nil {
					return nil, err
				}
			default:
				return nil, fmt.Errorf("unknown import type %q", kind)
			}
			return true, nil
		})
	})
	api.bind(object, "exportFile", func(_ js.Value, args []js.Value) any {
		return browserPromise(func() (any, error) {
			if len(args) != 2 || args[0].Type() != js.TypeString || args[1].Type() != js.TypeNumber {
				return nil, fmt.Errorf("export requires a type and slot")
			}
			kind, slot := args[0].String(), args[1].Int()
			var name string
			var data []byte
			var err error
			switch kind {
			case "score":
				name = "scoreth10.dat"
				data, err = g.ExportScore()
			case "cfg":
				name, data = "th10.cfg", g.Config.NativeBytes()
			case "replay":
				name, err = game.ReplayFilename(slot)
				if err == nil {
					data, err = g.ExportReplay(slot)
				}
			default:
				return nil, fmt.Errorf("unknown export type %q", kind)
			}
			if err != nil {
				return nil, err
			}
			array := js.Global().Get("Uint8Array").New(len(data))
			js.CopyBytesToJS(array, data)
			return map[string]any{"name": name, "data": array}, nil
		})
	})
	js.Global().Set("th10", object)
	return api
}

func importArguments(args []js.Value) (string, int, []byte, error) {
	if len(args) != 3 || args[0].Type() != js.TypeString || args[1].Type() != js.TypeNumber || args[2].Type() != js.TypeObject || !args[2].InstanceOf(js.Global().Get("Uint8Array")) {
		return "", 0, nil, fmt.Errorf("import requires a type, slot, and Uint8Array")
	}
	kind := args[0].String()
	limit := game.MaxReplayImportBytes
	if kind == "score" {
		limit = game.MaxScoreImportBytes
	} else if kind != "replay" {
		return "", 0, nil, fmt.Errorf("unknown import type %q", kind)
	}
	length := args[2].Get("byteLength").Int()
	if length < 0 || length > limit {
		return "", 0, nil, fmt.Errorf("import exceeds %d bytes", limit)
	}
	data := make([]byte, length)
	js.CopyBytesToGo(data, args[2])
	return kind, args[1].Int(), data, nil
}

func (api *browserAPI) bind(object js.Value, name string, body func(js.Value, []js.Value) any) {
	f := js.FuncOf(body)
	api.functions = append(api.functions, f)
	object.Set(name, f.Value)
}

func (api *browserAPI) close() {
	js.Global().Set("th10", js.Undefined())
	for _, function := range api.functions {
		function.Release()
	}
}

func browserPromise(work func() (any, error)) js.Value {
	executor := js.FuncOf(func(_ js.Value, args []js.Value) any {
		resolve, reject := args[0], args[1]
		// An IndexedDB wait inside js.FuncOf itself would deadlock its event loop.
		go func() {
			result, err := work()
			if err != nil {
				reject.Invoke(js.Global().Get("Error").New(err.Error()))
			} else {
				resolve.Invoke(result)
			}
		}()
		return nil
	})
	promise := js.Global().Get("Promise").New(executor)
	executor.Release()
	return promise
}

func platformMessage(name string, value any) {
	if listener := js.Global().Get(name); listener.Type() == js.TypeFunction {
		listener.Invoke(value)
	}
}

func metricsObject(m runtimeMetrics) map[string]any {
	return map[string]any{"tps": m.TPS, "fps": m.FPS, "updateLastMs": m.UpdateMS, "updateMeanMs": m.UpdateMeanMS, "updateMaxMs": m.UpdateMaxMS,
		"drawLastMs": m.DrawMS, "drawMeanMs": m.DrawMeanMS, "drawMaxMs": m.DrawMaxMS, "goHeapBytes": float64(m.HeapBytes)}
}
