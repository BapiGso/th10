package game

import "testing"

func TestRankingInsertTieAndShift(t *testing.T) {
	g := &Game{Save: DefaultSaveData()}
	state := NewState(1, 2, 3)
	state.Stage, state.Score, state.ReplayState90 = 6, 5000, 2
	table := g.ScoreTable(state)
	for i := range table {
		table[i] = ScoreEntry{Score: int32(1000 - i*100), Name: "OLD"}
	}
	at := g.InsertRanking(state, true, 1234, 0.5)
	if at != 5 || table[5].Score != 500 || table[6].Score != 500 || table[5].Timestamp != 1234 || table[5].Stage != 8 || table[5].Continues != 2 {
		t.Fatalf("insert/shift differs: at %d table %+v", at, table)
	}
	state.Score = 0
	if g.InsertRanking(state, false, 0, 0) != -1 {
		t.Fatal("non-ranking score inserted")
	}
}
