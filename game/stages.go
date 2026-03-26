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
