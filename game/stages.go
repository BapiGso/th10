package game

// StageInfo 关卡信息
type StageInfo struct {
	Number   int    // 关卡编号 (1-6, 7=Extra)
	Name     string // 日文名称
	Subtitle string // 副标题
	BGM      int    // 道中 BGM 编号 (audio.BGMStageN)
	BossBGM  int    // Boss BGM 编号
	MidBoss  string // 中Boss 名称（空=无中Boss）
	Boss     string // Boss 名称
}

// TH10Stages 风神录全关卡信息
var TH10Stages = [7]StageInfo{
	{
		Number: 1, Name: "秋の参道", Subtitle: "厄つきの参道",
		BGM: 1, BossBGM: 2,
		MidBoss: "秋静葉", Boss: "秋穣子",
	},
	{
		Number: 2, Name: "旧い参道の階段", Subtitle: "陰の参道",
		BGM: 3, BossBGM: 4,
		MidBoss: "", Boss: "鍵山雛",
	},
	{
		Number: 3, Name: "妖怪の住む山", Subtitle: "神々の息吹",
		BGM: 5, BossBGM: 6,
		MidBoss: "河城にとり", Boss: "河城にとり",
	},
	{
		Number: 4, Name: "秋暮の滝", Subtitle: "紅葉の河",
		BGM: 7, BossBGM: 8,
		MidBoss: "犬走椛", Boss: "射命丸文",
	},
	{
		Number: 5, Name: "妖怪の山の上", Subtitle: "守矢の境内",
		BGM: 9, BossBGM: 10,
		MidBoss: "", Boss: "東風谷早苗",
	},
	{
		Number: 6, Name: "妖怪の山 最奥", Subtitle: "信仰の頂",
		BGM: 11, BossBGM: 12,
		MidBoss: "", Boss: "八坂神奈子",
	},
	{
		Number: 7, Name: "Extra Stage", Subtitle: "妖怪の山の裏",
		BGM: 13, BossBGM: 14,
		MidBoss: "東風谷早苗", Boss: "洩矢諏訪子",
	},
}

// 文本标签 ----------------------------------------------------------------

var (
	characterNames  = [2]string{"博丽灵梦", "雾雨魔理沙"}
	characterTitles = [2]string{"乐园的巫女", "普通的魔法使"}

	// 短名（HUD / 存档摘要用）
	shotNames = [2][3]string{
		{"灵符", "梦符", "神符"},
		{"魔符", "恋符", "星符"},
	}
	// 长名（选机界面专用，含弹幕风格说明）
	shotDescriptions = [2][3]string{
		{"灵符（诱导弹）", "梦符（前方集中）", "神符（巫女敏符）"},
		{"魔符（魔法导弹）", "恋符（Master Spark）", "星符（星屑光线）"},
	}

	difficultyNames = [5]string{"Easy", "Normal", "Hard", "Lunatic", "Extra"}
)

func CharacterName(id int) string {
	if id < 0 || id >= len(characterNames) {
		return "未知角色"
	}
	return characterNames[id]
}

// CharacterTitle 角色称号（"乐园的巫女" 等），仅选机界面用。
func CharacterTitle(id int) string {
	if id < 0 || id >= len(characterTitles) {
		return ""
	}
	return characterTitles[id]
}

func ShotName(character, shotType int) string {
	if character < 0 || character >= len(shotNames) {
		return "未知装备"
	}
	row := shotNames[character]
	if shotType < 0 || shotType >= len(row) {
		return "未知装备"
	}
	return row[shotType]
}

// ShotDescription 装备长名（含弹幕风格描述），选机界面用。
func ShotDescription(character, shotType int) string {
	if character < 0 || character >= len(shotDescriptions) {
		return "未知装备"
	}
	row := shotDescriptions[character]
	if shotType < 0 || shotType >= len(row) {
		return "未知装备"
	}
	return row[shotType]
}

func DifficultyName(id int) string {
	if id < 0 || id >= len(difficultyNames) {
		return "Unknown"
	}
	return difficultyNames[id]
}

func StageName(stage int) string {
	if stage >= 1 && stage <= len(TH10Stages) {
		return TH10Stages[stage-1].Name
	}
	return "Unknown Stage"
}
