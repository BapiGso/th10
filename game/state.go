package game

// 难度常量
const (
	DiffEasy   = 0
	DiffNormal = 1
	DiffHard   = 2
	DiffLuna   = 3
	DiffExtra  = 4
)

// 角色常量
const (
	CharReimu  = 0 // 博丽灵梦
	CharMarisa = 1 // 雾雨魔理沙
)

// 装备常量（每角色 3 种）
const (
	ShotA = 0 // 灵梦:灵符 / 魔理沙:魔符
	ShotB = 1 // 灵梦:梦符 / 魔理沙:恋符
	ShotC = 2 // 灵梦:神符 / 魔理沙:星符
)

// 总关卡数
const (
	StageCount = 6 // 六面（1~6）
	StageExtra = 7 // Extra 面
)

// Power uses hundredths of displayed power here. The original stores units
// of 0.05; see docs/player_truth.md for the binary-to-project conversion.
const (
	PowerMax        = 500
	BombPowerCost   = 100
	DeathPowerLoss  = 320
	SmallPowerValue = 5
	BigPowerValue   = 100
)

// GameState 一局游戏的共享状态，Stage 场景持有，HUD 读取绘制
type GameState struct {
	Score         int64
	HiScore       int64
	Life          int // 剩余残机
	Power         int // 火力 0-500（显示为 0.00~5.00）
	MaxPower      int
	Point         int   // 得点
	Faith         int64 // 信仰值（影响得分倍率）
	MaxFaith      int64 // 信仰值上限
	Difficulty    int   // 0=Easy 1=Normal 2=Hard 3=Lunatic 4=Extra
	Character     int   // 0=灵梦 1=魔理沙
	ShotType      int   // 0=A 1=B 2=C
	Stage         int   // 当前关卡编号（1~6, 7=Extra）
	Frame         int   // 当前关卡已运行帧数
	Paused        bool
	ExtraUnlocked bool
	SingleStage   bool
	DebugMode     bool
	// RNGSeed seeds the shared RNG (DAT_004918b0) when HasSeed is set; a
	// replay chapter restores it from its header (+0x02).
	RNGSeed uint16
	HasSeed bool
	// Rank is the dynamic rank (DAT_00474c98, -1024..1024). It carries over
	// between stages; a replay chapter restores it from its header (+0x20).
	Rank int
	// The faith gauge block (DAT_00474c40 +0x10..+0x1c). FaithRate is the
	// stored faith lost on each decaying frame (DAT_00474c50, +0x10).
	// FaithHold is the hold timer (DAT_00474c58, +0x18): frames the faith
	// keeps its value before it decays. faithAcc is the timer's float
	// accumulator (DAT_00474c5c, +0x1c), whose truncation is the timer.
	FaithRate int
	FaithHold int
	faithAcc  float32
}

func NewState(character, shotType, difficulty int) *GameState {
	return &GameState{
		Life:       2,
		Power:      0,
		MaxPower:   PowerMax,
		Faith:      50000,
		MaxFaith:   999990, // 99999 in the binary
		Difficulty: difficulty,
		Character:  character,
		ShotType:   shotType,
		HiScore:    0,
	}
}

// AddScore 加分（受信仰倍率影响）
func (s *GameState) AddScore(v int64) {
	if v > 0 {
		v = int64(float64(v)*s.FaithMultiplier() + 0.5)
	}
	s.AddScoreRaw(v)
}

// AddScoreRaw adds an already-valued reward without applying the legacy faith
// multiplier. Point items use faith as their value, not as a second multiplier.
func (s *GameState) AddScoreRaw(v int64) {
	s.Score += v
	if s.Score > s.HiScore {
		s.HiScore = s.Score
	}
}

// AddPoints is FUN_00409d90: a tenth of n to the stored score, which shows
// times ten, capped at 999999999.
func (s *GameState) AddPoints(n int64) {
	s.Score = min(s.Score+n/10*10, 9999999990)
	if s.Score > s.HiScore {
		s.HiScore = s.Score
	}
}

// GainFaith is FUN_0041be80: a tenth of n to the faith (DAT_00474c4c, which
// this state keeps x10), capped at 99999.
func (s *GameState) GainFaith(n int64) {
	s.Faith += n / 10 * 10
	if s.Faith/10 > 99999 {
		s.Faith = 999990
	}
}

// GainPower is FUN_00418930: n units of 0.05 power (DAT_00474c48, which this
// state keeps x5) unless already full, capped at 1.00 x5. It reports whether
// the power level, one per 1.00, changed.
func (s *GameState) GainPower(n int) bool {
	p := s.Power / 5
	if p > 99 {
		return false
	}
	p = min(p+n, 100)
	s.Power = p * 5
	return (p-n)/20 != p/20
}

// LoseFaith is FUN_00412e70: a tenth of n off the faith (DAT_00474c4c, which
// this state keeps x10), floored at 5000.
func (s *GameState) LoseFaith(n int64) {
	s.Faith = max(s.Faith/10-n/10, 5000) * 10
}

// AddRank is FUN_00405b60: n to the rank, held to -0x400..0x400.
func (s *GameState) AddRank(n int) {
	s.Rank = min(max(s.Rank+n, -0x400), 0x400)
}

// AddFaith 增加信仰值
func (s *GameState) AddFaith(v int64) {
	s.Faith += v
	if s.Faith < 0 {
		s.Faith = 0
	}
	if s.Faith > s.MaxFaith {
		s.Faith = s.MaxFaith
	}
}

// Faith gauge constants. faithHoldCap is FUN_00412ff0's cap on the hold timer
// (0x82 = 130). faithDecayFloor is the stored floor of the decay (FUN_00418a00,
// 0x1388 = 5000). faithRateMin is the rate's floor once a decay step has run
// (FUN_00418a00, 0x12). faithRateStage is what FUN_00417870 sets at stage start.
const (
	faithHoldCap    = 130
	faithDecayFloor = 5000
	faithRateMin    = 18
	faithRateStage  = 9
)

// BeginStageFaith is the faith part of FUN_00417870 (stage start): DAT_00474c50
// becomes 9 before the stage loop starts.
func (s *GameState) BeginStageFaith() {
	s.FaithRate = faithRateStage
}

// AddFaithHold is FUN_00412ff0(n): n frames onto the hold timer, unless it is
// already at the cap (FUN_00412ff0 tests timer < 0x82 first); past the cap the
// timer is held at 130. FUN_0044bf40 adds n to the accumulator on its fast
// path (the game speed DAT_00476f78 is always 1.0, so it is 0.99 < speed <
// 1.01), and FUN_00463b2c truncates the accumulator to the timer.
func (s *GameState) AddFaithHold(n int) {
	if s.FaithHold >= faithHoldCap {
		return
	}
	s.faithAcc += float32(n)
	s.FaithHold = int(s.faithAcc)
	if s.FaithHold > faithHoldCap {
		s.FaithHold = faithHoldCap
		s.faithAcc = faithHoldCap
	}
}

// RestoreFaithHold sets the hold timer from a replay chapter: FUN_0042a6a0
// takes the chapter's +0x18 as DAT_00474c58 and its float as the accumulator.
func (s *GameState) RestoreFaithHold(n int) {
	s.FaithHold = n
	s.faithAcc = float32(n)
}

// TickFaith is FUN_00418a00, run once a frame when the stage's gate is open
// (scene/stage faithGate). While the hold timer is positive it counts down a
// frame (FUN_0044bf40 with -1.0). Otherwise, if the stored faith is above
// 5000, it loses FaithRate, the rate rises to at least 18, and the stored
// faith is held at no less than 5000.
func (s *GameState) TickFaith() {
	if s.FaithHold > 0 {
		s.faithAcc += -1
		s.FaithHold = int(s.faithAcc)
		return
	}
	stored := s.Faith / 10
	if stored <= faithDecayFloor {
		return
	}
	rate := s.FaithRate
	stored -= int64(rate)
	if rate < faithRateMin {
		s.FaithRate = faithRateMin
	}
	s.Faith = max(stored, faithDecayFloor) * 10
}

// FaithMultiplier 信仰倍率（用于得分计算）
func (s *GameState) FaithMultiplier() float64 {
	return float64(s.Faith) / 50000.0
}

// IsExtraUnlocked 是否解锁 Extra 面（需通关 Normal 以上）
func (s *GameState) IsExtraUnlocked() bool {
	return s.ExtraUnlocked
}

// NextStage 返回下一关编号，Stage 6 之后返回 0 表示通关
func (s *GameState) NextStage() int {
	if s.Stage >= StageCount {
		return 0 // 六面通关
	}
	return s.Stage + 1
}
