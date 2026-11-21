package main

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

type fakeRuntimeGame struct{ updates, draws int }

func (g *fakeRuntimeGame) Update() error            { g.updates++; return nil }
func (g *fakeRuntimeGame) Draw(*ebiten.Image)       { g.draws++ }
func (*fakeRuntimeGame) Layout(int, int) (int, int) { return 640, 480 }

func TestFirstGameFrameWaitsForGestureAndAudio(t *testing.T) {
	inner := &fakeRuntimeGame{}
	ready := false
	announced, started := 0, 0
	g := &gatedGame{inner: inner, audioReady: func() bool { return ready }, onReady: func() { announced++ }, onStart: func() { started++ }}
	if err := g.gate.request(); err == nil {
		t.Fatal("accepted a start before the audio initialization hook")
	}
	for range 100 {
		if err := g.Update(); err != nil {
			t.Fatal(err)
		}
		g.Draw(nil)
	}
	if inner.updates != 0 || inner.draws != 0 || announced != 1 {
		t.Fatal("scene calc/draw ran behind the gate")
	}
	if err := g.gate.request(); err != nil {
		t.Fatal(err)
	}
	if err := g.Update(); err != nil {
		t.Fatal(err)
	}
	g.Draw(nil)
	if inner.updates != 0 || inner.draws != 0 {
		t.Fatal("a gesture without audio readiness advanced the sig/title")
	}
	ready = true
	if err := g.Update(); err != nil {
		t.Fatal(err)
	}
	g.Draw(nil)
	if inner.updates != 1 || inner.draws != 1 || started != 1 {
		t.Fatal("first scene frame did not begin exactly once after unlocking")
	}
	if err := g.Update(); err != nil {
		t.Fatal(err)
	}
	if started != 1 || inner.updates != 2 {
		t.Fatal("start reset the original game more than once")
	}
}

func TestImportsCannotRaceTheFirstFrame(t *testing.T) {
	var gate startGate
	gate.initialize()
	if err := gate.beginImport(); err != nil {
		t.Fatal(err)
	}
	if gate.request() == nil || gate.beginImport() == nil || gate.tryStart(true) {
		t.Fatal("start or a second import raced a pending transaction")
	}
	gate.endImport()
	if err := gate.request(); err != nil {
		t.Fatal(err)
	}
	if gate.beginImport() == nil || !gate.tryStart(true) || gate.beginImport() == nil {
		t.Fatal("an import was accepted after starting the game")
	}
}

func TestStartupClockResumesAtAnEmptyDrawBoundary(t *testing.T) {
	inner := &fakeRuntimeGame{}
	resumed, started := 0, 0
	g := &gatedGame{inner: inner, audioReady: func() bool { return true }, waitForDraw: true,
		resumeClock: func() { resumed++ }, onStart: func() { started++ }}
	if err := g.Update(); err != nil {
		t.Fatal(err)
	}
	if err := g.gate.request(); err != nil {
		t.Fatal(err)
	}
	g.Draw(nil)
	if resumed != 0 || inner.draws != 0 {
		t.Fatal("startup did not first observe a zero-TPS draw")
	}
	g.Draw(nil)
	if resumed != 1 || inner.updates != 0 || inner.draws != 0 {
		t.Fatal("clock resume advanced a real frame before Update")
	}
	if err := g.Update(); err != nil {
		t.Fatal(err)
	}
	g.Draw(nil)
	if started != 1 || resumed != 1 || inner.updates != 1 || inner.draws != 1 {
		t.Fatal("bootstrap did not produce one first Update and Draw")
	}
}

func TestDebugToggleDoesNotStartOrAdvanceGame(t *testing.T) {
	inner := &fakeRuntimeGame{}
	g := &gatedGame{inner: inner, audioReady: func() bool { return false }}
	if !g.toggleDebug() || !g.debugEnabled() || g.toggleDebug() || g.debugEnabled() {
		t.Fatal("debug toggle state differs")
	}
	if g.gate.pendingStart() || g.gate.isStarted() || inner.updates != 0 || inner.draws != 0 {
		t.Fatal("debug toggle changed the game or startup gate")
	}
}
