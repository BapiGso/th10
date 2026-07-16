package title

import (
	"image/color"
	"th10/audio"
	"th10/game"
	"th10/input"
	"th10/render"
	"th10/scene/menus"
	selector "th10/scene/select"
	sceneui "th10/scene/ui"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	menuStart = iota
	menuExtraStart
	menuPractice
	menuReplay
	menuPlayerData
	menuMusicRoom
	menuOption
	menuQuit
	menuCount
)

var menuLabels = [menuCount]string{
	"Game Start",
	"Extra Start",
	"Practice Start",
	"Replay",
	"Player Data",
	"Music Room",
	"Option",
	"Quit",
}

// 菜单文字配色 — 参照原版 TH10 题图（.tmp/img_2.jpg）。
var (
	menuIdleColor     = color.NRGBA{69, 43, 44, 255}    // 暗酒红：未选中
	menuHoverColor    = color.NRGBA{255, 163, 163, 255} // 亮粉红：选中（hover/focus）
	menuOutlineColor  = color.NRGBA{0, 0, 0, 235}       // 黑色描边
	menuDisabledColor = color.NRGBA{80, 70, 70, 220}    // Extra 锁住时
	hintColor         = color.NRGBA{228, 222, 214, 240}
)

const (
	menuStepX        = 8  // 阶梯：每往下一行，文字右缘再向右挪 8px
	menuLineH        = 30 // 行高
	menuRightMargin  = 24 // 最末项右缘距屏幕右边缘的留白
	menuVerticalBias = 40 // 整组菜单相对屏幕中心的纵向偏移
)

// Title 标题场景。
//
// 这里的菜单不直接走 ebitenui Button —— 我们要给文字加八方向描边，
// 而 ebitenui Button 没有 outline 选项；通过 sceneui.MenuList 走自绘路径。
type Title struct {
	frame       int
	menu        *sceneui.MenuList
	extraUnlock bool
	bgmStarted  bool
	bgImg       *ebiten.Image
	quit        bool
}

func New() *Title {
	t := &Title{}
	// title00s.png 是完整的静态标题画面（已含 東方風神録 / Mountain of Faith
	// 文字、巫女与背景），所以不再单独叠加 title_logo.png — 否则会出现重复 logo。
	t.bgImg = render.LoadImage("anm/title/title00s.png")
	return t
}

func (t *Title) buildMenu(g *game.Game) *sceneui.MenuList {
	itemFace := render.LoadSystemFace(22)

	items := make([]sceneui.MenuItem, menuCount)
	for i := 0; i < menuCount; i++ {
		items[i] = sceneui.MenuItem{Label: labelOf(i, t.extraUnlock)}
	}
	if !t.extraUnlock {
		items[menuExtraStart].Disabled = true
	}

	// 计算 baseX：让"最远右伸"那行也能贴近右边距而不出屏
	maxExtent := 0
	for i := 0; i < menuCount; i++ {
		w, _ := render.MeasureText(itemFace, items[i].Label)
		ext := i*menuStepX + w
		if ext > maxExtent {
			maxExtent = ext
		}
	}
	baseX := game.ScreenWidth - menuRightMargin - maxExtent
	totalH := menuCount * menuLineH
	topY := game.ScreenHeight/2 + menuVerticalBias - totalH/2

	menu := &sceneui.MenuList{
		Items:  items,
		X:      baseX,
		Y:      topY,
		LineH:  menuLineH,
		StepX:  menuStepX,
		Cursor: 0,
		Style: sceneui.MenuStyle{
			Face:     itemFace,
			Idle:     menuIdleColor,
			Hover:    menuHoverColor,
			Disabled: menuDisabledColor,
			Outline:  menuOutlineColor,
		},
		OnMove: func(int) {
			g.Audio().PlaySE(audio.SESelect)
		},
		OnActivate: func(idx int) {
			g.Audio().PlaySE(audio.SEOk)
			t.activate(g, idx)
		},
		// X 只把光标挪到 Quit；想退出还得再按 Z 确认（沿用旧行为）
		OnCancel: func() {
			menu := t.menu
			if menu != nil {
				menu.Cursor = menuQuit
			}
		},
	}
	return menu
}

func (t *Title) Update(g *game.Game) error {
	t.frame++
	t.extraUnlock = g.Save != nil && g.Save.ExtraUnlocked

	if !t.bgmStarted {
		g.Audio().PlayBGM(audio.BGMTitle)
		t.bgmStarted = true
	}

	// 第一次/解锁状态变化时重建菜单（label 含 "(Locked)" 字样会变）
	if t.menu == nil {
		t.menu = t.buildMenu(g)
	} else {
		// 只更新 label/disabled，不改 cursor，避免重置选中
		for i := 0; i < menuCount; i++ {
			t.menu.Items[i].Label = labelOf(i, t.extraUnlock)
			t.menu.Items[i].Disabled = (i == menuExtraStart && !t.extraUnlock)
		}
	}

	t.menu.Update()
	_ = input.Global // input 由 menu.Update 内部读取

	if t.quit {
		return ebiten.Termination
	}
	return nil
}

func (t *Title) activate(g *game.Game, i int) {
	switch i {
	case menuStart:
		g.GoTo(selector.New(t))
	case menuExtraStart:
		if t.extraUnlock {
			g.GoTo(selector.NewExtra(t))
		}
	case menuPractice:
		g.GoTo(selector.NewPractice(t))
	case menuReplay:
		g.GoTo(menus.NewReplay(t))
	case menuPlayerData:
		g.GoTo(menus.NewPlayerData(g.Save, t))
	case menuMusicRoom:
		g.GoTo(menus.NewMusicRoom(t))
	case menuOption:
		g.GoTo(menus.NewOption(g, t))
	case menuQuit:
		t.quit = true
	}
}

func (t *Title) Draw(screen *ebiten.Image) {
	if t.bgImg != nil {
		screen.DrawImage(t.bgImg, nil)
	} else {
		screen.Fill(color.RGBA{12, 8, 32, 255})
	}

	if t.menu != nil {
		t.menu.Draw(screen)
	}

	// 底部操作提示
	hintFace := render.LoadSystemFace(16)
	hint := "Arrow Keys: Move   Z: Select   X: Cancel"
	hw, _ := render.MeasureText(hintFace, hint)
	render.DrawText(screen, hint, hintFace, (game.ScreenWidth-hw)/2, game.ScreenHeight-30, hintColor)
}

// labelOf 返回某一项当前应显示的文字（处理 Extra 锁定）。
func labelOf(i int, extraUnlock bool) string {
	if i == menuExtraStart && !extraUnlock {
		return "Extra Start"
	}
	return menuLabels[i]
}
