//go:build js

package audio

import (
	"encoding/binary"
	"math"
	"syscall/js"

	"github.com/hajimehoshi/ebiten/v2/audio"
)

// webSE plays the voices through Web Audio. Ebiten's players are mixed in
// Go on the page's only thread, and every new player pre-reads half a
// second of PCM there, which made shot-heavy scenes stutter; here a play is
// a handful of JS calls and the browser mixes on its audio thread.
//
// Each id has a fixed graph, splitter -> two gains (left, right) -> merger
// -> master, made on its first play. A play stops the id's previous
// AudioBufferSourceNode (they are one-shot by design) and starts a new one
// on the same graph from the wav's AudioBuffer.
//
// The AudioContext is made on the first play, after the page's start gate
// (a user gesture), so the browser lets it run; all the wavs are uploaded
// then. Without Web Audio (Node) it stays silent.
type webSE struct {
	sampleRate int
	pcm        map[string][]byte
	ctx        js.Value
	master     js.Value
	volume     float64
	running    bool
	failed     bool
	buffers    map[*byte]js.Value // by &pcm[0]: one AudioBuffer per wav
	voices     [len(seTable)]webVoice
}

type webVoice struct {
	src, left, right, splitter js.Value
	ready                      bool
}

func newSEOutput(ctx *audio.Context, pcm map[string][]byte) seOutput {
	return &webSE{sampleRate: ctx.SampleRate(), pcm: pcm, volume: 1}
}

func (w *webSE) setMaster(v float64) {
	w.volume = v
	if w.master.Truthy() {
		w.master.Get("gain").Set("value", v)
	}
}

func (w *webSE) play(id int, pcm []byte, left, right float64) {
	if w.failed || len(pcm) < pcmBytesPerFrame {
		return
	}
	defer func() {
		if recover() != nil {
			w.failed = true // a JS exception: leave sound effects off, keep the game running
		}
	}()
	if !w.ctx.Truthy() && !w.open() {
		return
	}
	if !w.running {
		if w.ctx.Get("state").String() == "running" {
			w.running = true
		} else {
			w.ctx.Call("resume")
		}
	}
	buf, ok := w.buffers[&pcm[0]]
	if !ok {
		return
	}
	v := &w.voices[id]
	if !v.ready {
		v.splitter = w.ctx.Call("createChannelSplitter", 2)
		merger := w.ctx.Call("createChannelMerger", 2)
		v.left, v.right = w.ctx.Call("createGain"), w.ctx.Call("createGain")
		v.splitter.Call("connect", v.left, 0)
		v.splitter.Call("connect", v.right, 1)
		v.left.Call("connect", merger, 0, 0)
		v.right.Call("connect", merger, 0, 1)
		merger.Call("connect", w.master)
		v.ready = true
	}
	if v.src.Truthy() {
		v.src.Call("stop")
		v.src.Call("disconnect")
	}
	v.left.Get("gain").Set("value", left)
	v.right.Get("gain").Set("value", right)
	v.src = w.ctx.Call("createBufferSource")
	v.src.Set("buffer", buf)
	v.src.Call("connect", v.splitter)
	v.src.Call("start")
}

// open makes the context and the master gain and uploads every wav.
func (w *webSE) open() bool {
	class := js.Global().Get("AudioContext")
	if !class.Truthy() {
		class = js.Global().Get("webkitAudioContext")
	}
	if !class.Truthy() {
		w.failed = true
		return false
	}
	w.ctx = class.New(map[string]any{"latencyHint": "interactive"})
	w.master = w.ctx.Call("createGain")
	w.master.Get("gain").Set("value", w.volume)
	w.master.Call("connect", w.ctx.Get("destination"))
	w.resumeOnGesture()
	w.buffers = make(map[*byte]js.Value, len(w.pcm))
	for _, pcm := range w.pcm {
		if len(pcm) >= pcmBytesPerFrame {
			w.buffers[&pcm[0]] = w.upload(pcm)
		}
	}
	w.pcm = nil
	return true
}

// resumeOnGesture resumes a suspended context from the next input event.
// Chrome lets a context start after any earlier gesture on the page, but
// Safari only resumes inside a gesture handler, and a play runs in a frame.
func (w *webSE) resumeOnGesture() {
	doc := js.Global().Get("document")
	var resume js.Func
	resume = js.FuncOf(func(js.Value, []js.Value) any {
		if w.ctx.Get("state").String() != "running" {
			w.ctx.Call("resume")
			return nil
		}
		for _, ev := range []string{"keydown", "pointerdown", "touchend"} {
			doc.Call("removeEventListener", ev, resume, true)
		}
		resume.Release()
		return nil
	})
	for _, ev := range []string{"keydown", "pointerdown", "touchend"} {
		doc.Call("addEventListener", ev, resume, true)
	}
}

// upload copies 16-bit stereo PCM into a new stereo AudioBuffer, sample/32768.
func (w *webSE) upload(pcm []byte) js.Value {
	frames := len(pcm) / pcmBytesPerFrame
	buf := w.ctx.Call("createBuffer", 2, frames, w.sampleRate)
	raw := make([]byte, frames*4)
	array := js.Global().Get("Uint8Array").New(frames * 4)
	floats := js.Global().Get("Float32Array").New(array.Get("buffer"))
	for ch := range 2 {
		for f := range frames {
			s := int16(binary.LittleEndian.Uint16(pcm[f*pcmBytesPerFrame+ch*2:]))
			binary.LittleEndian.PutUint32(raw[f*4:], math.Float32bits(float32(s)/32768))
		}
		js.CopyBytesToJS(array, raw)
		buf.Call("copyToChannel", floats, ch)
	}
	return buf
}
