package dialog

// Line 字面量是 Speaker / Text / IsRight，stage 脚本通过 MidPre/BossPre/Post 取这里的对白。
// 资源被硬编码进二进制，无需附带任何 JSON。

func r(s, t string) Line { return Line{Speaker: s, Text: t, IsRight: true} }
func l(s, t string) Line { return Line{Speaker: s, Text: t, IsRight: false} }

const (
	reimu    = "灵梦"
	shizuha  = "秋静葉"
	minoriko = "秋穣子"
	hina     = "鍵山雛"
	nitori   = "河城にとり"
	momiji   = "犬走椛"
	aya      = "射命丸文"
	sanae    = "東風谷早苗"
	kanako   = "八坂神奈子"
	suwako   = "洩矢諏訪子"
)

type stageScript struct {
	midPre  []Line // 中BOSS战前
	bossPre []Line // 关底BOSS战前
	post    []Line // 关底BOSS被击败后
}

// 剧本来源：thbwiki《游戏对话:东方风神录/博丽灵梦/中日对照》及 ExStory 同名条目。
// 为了在游戏内对话框里显示通顺，部分长句改为更短的中文意译，剧情骨架与 wiki 一致。
var scripts = map[string]stageScript{
	"stage1": {
		midPre: []Line{
			l(reimu, "落叶纷飞搞得视野很不好呢。"),
			l(reimu, "这种情况下闯进山里会不会出问题？"),
			r(shizuha, "巫女啊，秋天的山可不是给你乱闯的。"),
			r(shizuha, "我是红叶之神秋静葉，给你点教训吧。"),
		},
		bossPre: []Line{
			l(reimu, "哎呀？有一种好吃东西的味道……"),
			r(minoriko, "身为巫女还妄想把神明当吃的啊。"),
			r(minoriko, "何等可笑、何等失态！"),
			l(reimu, "我啥时候说过要吃啦。"),
			l(reimu, "但是，这诱人的香味是你发出来的吗？"),
			r(minoriko, "身为神的我，当然也讲究身上的香气。"),
			r(minoriko, "啊，顺带一提我是丰收之神哦。"),
			l(reimu, "呜——嗯，是生的烤红薯的香气呢。"),
			r(minoriko, "丰登的红薯正是我的香水！"),
			r(minoriko, "就这样给巫女吃掉了怎么能行呢！"),
		},
		post: []Line{
			l(reimu, "哼，所谓神明也是优劣不等呢。"),
			l(reimu, "我家里的神应该不会比这样的还弱吧？"),
			l(reimu, "现在可不是烤红薯的时候，先赶路吧。"),
		},
	},
	"stage2": {
		bossPre: []Line{
			l(reimu, "从一开始就觉得很不舒服啊。"),
			l(reimu, "这地方空气很沉闷，明明白天却暗无天日。"),
			r(hina, "哎呀哎呀，你还在这啊？"),
			r(hina, "我刚才明明还费心赶你回去来着。"),
			l(reimu, "刚才？"),
			r(hina, "算了，我只是想把迷路的人类引上回头路。"),
			l(reimu, "我没迷路，只是想要去山上啦。"),
			r(hina, "一个人类去山里做什么，很危险的哦？"),
			l(reimu, "妨碍我的话，你就是我的敌人了哦？"),
			r(hina, "我是人类的朋友，会替你承受所有灾厄。"),
			r(hina, "你的灾厄要不要也全部交给我呢？"),
			l(reimu, "妖怪是敌人，而你就是妖怪……"),
			r(hina, "啊，这样啊！"),
		},
		post: []Line{
			r(hina, "我可是出于一番好意才要赶你走的……"),
			l(reimu, "把人赶走这种事情原本就谈不上好意。"),
			r(hina, "此路通往众神栖居之所。"),
			r(hina, "你会后悔的，那并非人类可以涉足之地。"),
			l(reimu, "啊，是么？接着总算能进入妖怪之山了。"),
		},
	},
	"stage3": {
		midPre: []Line{
			r(nitori, "哎哎，人类！？"),
			l(reimu, "等、等下？要去哪里啊？"),
		},
		bossPre: []Line{
			r(nitori, "哎呀，是刚才的人类。"),
			r(nitori, "我说了不要再往里走了不是吗？"),
			l(reimu, "你刚才还真敢阻挠我呢。"),
			r(nitori, "阻挠？我连你葫芦里卖什么药都不知道呢。"),
			l(reimu, "我有急事要找住在山上的神明谈谈。"),
			l(reimu, "让我过去行么？"),
			r(nitori, "山上的神明？那种东西可有好多个呢……"),
			r(nitori, "听我的话没错，你还是回去比较好。"),
			r(nitori, "我是河城にとり，俗称溪谷河童的にとり。"),
			r(nitori, "再往前走，会遇到很多排斥人类的家伙哦。"),
			l(reimu, "明知山有虎偏向虎山行，事出无奈啊。"),
			r(nitori, "拿你没辙，那就让我看看你的决心吧！"),
		},
		post: []Line{
			r(nitori, "好厉害，连我的兵器都打不倒……"),
			r(nitori, "身为人类却强得超乎想象。"),
			l(reimu, "那么，我先走一步喽。"),
			r(nitori, "山上确实住了不安分的神明。"),
			r(nitori, "你是去打倒她的吧？这一带我会去通知。"),
			l(reimu, "看见瀑布了……好戏现在才开始呢！"),
		},
	},
	"stage4": {
		midPre: []Line{
			l(reimu, "前面就是山顶了吧。"),
			r(momiji, "白狼天狗在此，闲人禁止通过。"),
			r(momiji, "胆敢闯入警戒线的人类，统统赶下山！"),
		},
		bossPre: []Line{
			r(aya, "啊呀呀呀呀。"),
			r(aya, "接到入侵者报告特意来瞧瞧——"),
			r(aya, "怎么是你这家伙啊……"),
			l(reimu, "我没事要找你们天狗办，让个路啦。"),
			r(aya, "明明我只是个新闻记者来的。"),
			r(aya, "对你的事最了解的，就是我了。"),
			l(reimu, "我要见住在山上的神明，你知道点什么吗？"),
			r(aya, "呵呵～也就是说那个神喽？"),
			r(aya, "最近住下了一个连天狗都觉得棘手的神。"),
			r(aya, "她还想把山脚为止的信仰全部收过去呢。"),
			l(reimu, "……在收集信仰？那就是她了。"),
			l(reimu, "我要会会她，她在哪里？"),
			r(aya, "天狗已经打算解决了，没必要放你插手。"),
			l(reimu, "我都跑到这里来了，有什么不好的。"),
			r(aya, "但我是不可以放你过去的。"),
			r(aya, "见回り天狗们可不会同意呢。"),
			l(reimu, "天狗这种族真是麻烦啊。"),
			r(aya, "我会手下留情，尽管放马过来吧！"),
		},
		post: []Line{
			r(aya, "你强得超出我的想象。"),
			r(aya, "这种程度，或许和那麻烦的神有得一搏呢。"),
			l(reimu, "那么，告诉我那个神在哪里！"),
			r(aya, "她前不久把神社和湖一起搬上了山。"),
			r(aya, "再往前会出现一座新神社，应该就在那里。"),
			l(reimu, "山上的神社？神社的话原本只有我一家啊……"),
		},
	},
	"stage5": {
		midPre: []Line{
			r(sanae, "巫女的你居然到山上来了……"),
			r(sanae, "莫非是忙着来恭迎我们家的神明吗？"),
			l(reimu, "原来这里真的是神社……"),
			l(reimu, "原来除了我家，其他神社也存在啊。"),
		},
		bossPre: []Line{
			r(sanae, "此处是守矢神社，被遗忘的往昔之神社。"),
			r(sanae, "连同湖一起，从外面世界搬到幻想乡来。"),
			l(reimu, "搬神社又搬湖，还真够排场的。"),
			r(sanae, "这座山由我和我的神明接收。"),
			r(sanae, "再得到你的神社的话——"),
			r(sanae, "幻想乡的信仰，我们将囊括在手……"),
			l(reimu, "你觉得幻想乡的八百万众神会让你？"),
			r(sanae, "这么做也是为了幻想乡。"),
			r(sanae, "若信仰持续缺失，幻想乡也会失去力量。"),
			r(sanae, "也将失去那引发奇迹的力量！"),
			l(reimu, "一派胡言！信仰心，我自己取回来！"),
			r(sanae, "我是风祝早苗，外界绝迹的现人神末裔。"),
			r(sanae, "祭祀神明的人也可成为被祭祀者。"),
			r(sanae, "你身为巫女，做好这种觉悟了吗？"),
			l(reimu, "成不成神我都无所谓。"),
			l(reimu, "若是要做的话，到时候再考虑就好！"),
			r(sanae, "那就在现人神的力量洗礼中思索吧！"),
			r(sanae, "这召唤奇迹的神明之力！"),
		},
		post: []Line{
			r(sanae, "好强……"),
			r(sanae, "拥有这般力量，为何你的神社聚不起信仰？"),
			l(reimu, "这个我也想知道。"),
			r(sanae, "把我家神明设个分社，信仰心也能回复不少。"),
			l(reimu, "嗯——这倒值得考虑。"),
			l(reimu, "但首先得见见那个神明。"),
			r(sanae, "诶！？你此行的目的难不成……"),
			l(reimu, "就是给惹是生非的神明一点惩治喽～"),
		},
	},
	"stage6": {
		bossPre: []Line{
			l(reimu, "抵达湖边了，应该就在这里。"),
			l(reimu, "这令人生厌的柱子山……快现身吧！"),
			r(kanako, "何人在呼唤我？"),
			r(kanako, "哎呀？这不是山脚的巫女么？"),
			r(kanako, "有何贵干？"),
			l(reimu, "还真是大大咧咧的神啊。"),
			r(kanako, "比起庄严，朋友感觉更易聚拢信仰。"),
			l(reimu, "你要夺我的神社，会造成困扰，能罢手吗？"),
			r(kanako, "我没想夺取，只是想帮你的神社一把。"),
			r(kanako, "让人类聚到你的神社，免于妖怪的魔手。"),
			l(reimu, "多管闲事呢。"),
			l(reimu, "再说祭祀你信仰会增加吗？这是个未知数。"),
			r(kanako, "信仰绝不会低于零。"),
			r(kanako, "幻想乡所缺的，正是对神明的信仰心。"),
			r(kanako, "作为巫女的你，应当明白吧？"),
			l(reimu, "我也想看到神社里有参拜客。"),
			l(reimu, "但那要靠我自己——不借你的力量！"),
			r(kanako, "神社不是为巫女而存在的。"),
			r(kanako, "神社乃神明栖居之地——也差不多——"),
			r(kanako, "该让你认真考虑神社存在的意义了！"),
		},
		post: []Line{
			r(kanako, "看来你的力量配得上这片土地。"),
			r(kanako, "信仰不是枷锁，是人与神之间的契约。"),
			r(kanako, "只要还有人相信，神社就永远不会倒塌。"),
			l(reimu, "信仰心啊……我也得好好想想了。"),
		},
	},
	"extra": {
		midPre: []Line{
			r(kanako, "哎呀，莫非你还要继续前行？"),
			r(kanako, "不行哦，因为我的永眠之友在这里呢。"),
		},
		bossPre: []Line{
			l(reimu, "神奈子说的那个朋友……"),
			l(reimu, "果然也是一个神明吧。"),
			r(suwako, "谁和她是朋友啊？"),
			r(suwako, "自说自话把我神社搬进幻想乡——"),
			r(suwako, "事到如今还胡言乱语。那女的是敌人啊敌人。"),
			l(reimu, "这神社不是神奈子的吗？"),
			r(suwako, "啊——呜——本来是我的没错啦。"),
			l(reimu, "本来？"),
			r(suwako, "过去败给神奈子后，就成了她的神社。"),
			r(suwako, "不过她允我自由来去，信仰也增加了。"),
			r(suwako, "对她也算不上没有感谢之情。"),
			r(suwako, "话说回来，你是山下的巫女吧？"),
			l(reimu, "我只是来探探这神社有什么秘密。"),
			l(reimu, "你们俩关系挺好的，那我该回去了吧？"),
			r(suwako, "你说什么啊。"),
			r(suwako, "都和早苗、神奈子打过了——"),
			r(suwako, "只无视我一个，你这巫女也做得出？"),
			l(reimu, "是这么觉得。"),
			r(suwako, "真是的！是巫女的话就听好了！"),
			r(suwako, "「祭典」别名「神遊」，神和人类一起玩！"),
			l(reimu, "之前跟早苗、神奈子的战斗也是……？"),
			r(suwako, "正是歌舞奉神，也就是祭典！"),
			r(suwako, "今天轮到我主办的弹幕祭典啦！"),
		},
		post: []Line{
			r(suwako, "啊哈哈哈，干得漂亮。"),
			r(suwako, "建立过一个国家的我，竟然输给人类。"),
			l(reimu, "这算哪门子祭典，都是弹幕嘛。"),
			r(suwako, "祭典就是神和人一起玩，是有别于日常的仪式。"),
			l(reimu, "呃——弹幕本来就很有日常气息了。"),
			r(suwako, "有这么强的人，住在幻想乡也不赖。"),
			r(suwako, "你的神社也办个祭典吧，定个例大祭。"),
			l(reimu, "弹幕祭典？真的会有人来吗？"),
			r(suwako, "我是制品部，神奈子是销售部——"),
			r(suwako, "神奈子的神德和信仰心，多半源自我的力量。"),
			l(reimu, "神的世界也是处事艰难呢……"),
		},
	},
}

func cloneLines(src []Line) []Line {
	if len(src) == 0 {
		return nil
	}
	out := make([]Line, len(src))
	copy(out, src)
	return out
}

// MidPre 中BOSS战前对话（无中BOSS的关返回 nil）。
func MidPre(stage string) []Line { return cloneLines(scripts[stage].midPre) }

// BossPre 关底BOSS战前对话。
func BossPre(stage string) []Line { return cloneLines(scripts[stage].bossPre) }

// Post 关底BOSS被击败后对话。
func Post(stage string) []Line { return cloneLines(scripts[stage].post) }

// --- 官方 .msg 版（优先）：按角色加载原版对话，缺失时回退到手写表 ---

// msgSegment 返回某关某角色的第 seg 段官方对话（从 0 起）。无则 nil。
func msgSegment(stage string, character, seg int) []Line {
	sd := loadStageDialog(stage, character)
	if sd == nil || seg < 0 || seg >= len(sd.segments) {
		return nil
	}
	return cloneLines(sd.segments[seg])
}

// segmentLayout 描述某关 .msg 段顺序到 (midPre,bossPre,post) 的映射；-1=该段不存在。
// 多数关 [bossPre, post]；stage3 有中Boss = [midPre, midPost(忽略), bossPre, post]。
type segmentLayout struct{ midPre, bossPre, post int }

var msgLayouts = map[string]segmentLayout{
	"stage1": {midPre: 0, bossPre: 1, post: -1}, // 秋姉妹: 静葉(中), 穣子(底)
	"stage2": {midPre: -1, bossPre: 0, post: 1},
	"stage3": {midPre: 0, bossPre: 2, post: 3},
	"stage4": {midPre: -1, bossPre: 0, post: 1},
	"stage5": {midPre: -1, bossPre: 0, post: 1},
	"stage6": {midPre: -1, bossPre: 0, post: 1},
	"extra":  {midPre: 0, bossPre: 1, post: 2},
}

// MidPreC / BossPreC / PostC：角色感知的对话获取。优先官方 .msg，回退手写表。
func MidPreC(stage string, character int) []Line {
	if l, ok := msgLayouts[stage]; ok && l.midPre >= 0 {
		if seg := msgSegment(stage, character, l.midPre); seg != nil {
			return seg
		}
	}
	return MidPre(stage)
}

func BossPreC(stage string, character int) []Line {
	if l, ok := msgLayouts[stage]; ok && l.bossPre >= 0 {
		if seg := msgSegment(stage, character, l.bossPre); seg != nil {
			return seg
		}
	}
	return BossPre(stage)
}

func PostC(stage string, character int) []Line {
	if l, ok := msgLayouts[stage]; ok && l.post >= 0 {
		if seg := msgSegment(stage, character, l.post); seg != nil {
			return seg
		}
	}
	return Post(stage)
}
