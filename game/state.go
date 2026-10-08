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
	Graze         int
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
}

func NewState(character, shotType, difficulty int) *GameState {
	return &GameState{
		Life:       2,
		Power:      0,
		MaxPower:   PowerMax,
		Faith:      50000,
		MaxFaith:   100000,
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

// DecayFaith 信仰值自然衰减（每帧调用）
func (s *GameState) DecayFaith() {
	decay := int64(1) // 每帧衰减1
	s.Faith -= decay
	if s.Faith < 0 {
		s.Faith = 0
	}
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
