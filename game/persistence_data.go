package game

// SaveData is the host view of DAT_0047783c. The JSON tags are read only for
// migration; all new progress writes use ScoreFile's native CR/ST container.
type SaveData struct {
	Native           *ScoreFile           `json:"-"`
	SpellHistory     *SpellHistory        `json:"-"`
	Progress         ScoreProgress        `json:"-"`
	LoadError        string               `json:"-"`
	HighScore        int64                `json:"high_score"`
	ExtraUnlocked    bool                 `json:"extra_unlocked"`
	TotalRuns        int                  `json:"total_runs"`
	TotalClears      int                  `json:"total_clears"`
	TotalExtraClears int                  `json:"total_extra_clears"`
	TotalGameOvers   int                  `json:"total_game_overs"`
	LastCharacter    int                  `json:"last_character"`
	LastShotType     int                  `json:"last_shot_type"`
	LastDifficulty   int                  `json:"last_difficulty"`
	LastReplayName   string               `json:"last_replay_name,omitempty"`
	Rankings         [6][5][10]ScoreEntry `json:"rankings,omitempty"`
}

// ScoreEntry is FUN_00421fa0's 0x18-byte ranking record: stored score +0,
// stage +4, continues (the displayed final digit) +5, name +6 (nine bytes
// including NUL), time32 +0x10, slow rate +0x14. The native per-shot block
// begins at +8 and has stride 0x437c; its rankings begin at relative +0x10.
type ScoreEntry struct {
	Score     int32      `json:"score"`
	Stage     uint8      `json:"stage"`
	Continues uint8      `json:"continues"`
	Name      string     `json:"name"`
	Timestamp int32      `json:"timestamp"`
	SlowRate  float32    `json:"slow_rate"`
	Raw       [0x18]byte `json:"-"` // preserve padding/name tails while moving all six original dwords
}

func (g *Game) ScoreTable(state *GameState) *[10]ScoreEntry {
	if g.Save == nil {
		g.Save = DefaultSaveData()
	}
	shot := min(max(state.Character*3+state.ShotType, 0), 5)
	difficulty := min(max(state.Difficulty, 0), 4)
	return &g.Save.Rankings[shot][difficulty]
}

// InsertRanking ports FUN_00421fa0. A tie is inserted before the old entry
// (0x421fc2 JLE), and all six dwords of lower entries move down one slot.
func (g *Game) InsertRanking(state *GameState, clear bool, stamp int32, slow float32) int {
	table := g.ScoreTable(state)
	score := int32(state.Score / 10)
	at := 0
	for at < len(table) && table[at].Score > score {
		at++
	}
	if at == len(table) {
		return -1
	}
	copy(table[at+1:], table[at:len(table)-1])
	stage := state.Stage
	if clear {
		stage = 8 // FUN_00423570 temporarily uses 8 for an Extra clear
	}
	table[at] = ScoreEntry{
		Score: score, Stage: uint8(stage), Continues: uint8(state.ReplayState90),
		Name: "        ", Timestamp: stamp, SlowRate: slow,
	}
	return at
}
