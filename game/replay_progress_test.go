package game

import (
	"reflect"
	"testing"
)

func TestPlaybackNeverMarksProgress(t *testing.T) {
	g := &Game{Save: DefaultSaveData()}
	g.Save.TotalRuns, g.Save.TotalClears, g.Save.TotalGameOvers, g.Save.HighScore = 4, 3, 2, 100
	want := *g.Save
	state := NewState(0, 0, DiffNormal)
	state.ReplayMode, state.Stage, state.Score = 1, StageCount, 999999990
	g.BeginRun(state)
	g.MarkGameOver(state)
	g.MarkClear(state)
	if !reflect.DeepEqual(*g.Save, want) {
		t.Fatal("replay changed progress")
	}
}
