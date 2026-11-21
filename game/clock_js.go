//go:build js

package game

func platformTPS() int { return 60 }

func (*Game) logicFrameReady(float64) bool { return true }
