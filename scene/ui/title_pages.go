package ui

import (
	"fmt"
	"math"

	"th10/audio"
	"th10/game"
	"th10/input"
	"th10/supervisor"
)

// SetRunHooks leaves run/playback construction with the scene packages.
func (c *TitleController) SetRunHooks(start func(*game.Game, *game.GameState) error, demo func(*game.Game, int) error) {
	c.onStart, c.onDemo = start, demo
}

// BeginExtra is FUN_0042d420 at 0x42d785: remember the old difficulty.
func (c *TitleController) BeginExtra() { c.oldDifficulty, c.Difficulty = c.Difficulty, game.DiffExtra }

func (c *TitleController) Remember(g *game.Game) {
	g.Selection.Character, g.Selection.Shot, g.Selection.Difficulty = c.Character, c.Shot, c.Difficulty
	g.Selection.PracticeStage = c.PracticeStage
}

func (c *TitleController) playSE(g *game.Game, id int) {
	if g.Audio() != nil {
		g.Audio().PlaySEID(id)
	} // FUN_0043dc90, centered menu sound
}

// ExtraAvailable is FUN_0042c840/0042c850: ST +0x1c..+0x21, not the
// host's aggregate ExtraUnlocked. Ending owns writes to these bytes (t11).
func ExtraAvailable(save *game.SaveData, shot int) bool {
	return save != nil && save.Native != nil && shot >= 0 && shot < 6 && save.Native.Stats[0x1c+shot] != 0
}

// calcPage is FUN_0042cdf0's owned-page switch. Idle counts the current
// held mask, even during intro; direction/focus keys do not reset it.
func (c *TitleController) calcPage(g *game.Game, in *input.State) error {
	m := in.Menu()
	if c.State == TitleIntro || c.State == TitleMain {
		if m.Current&input.MenuIdleReset != 0 {
			c.Idle = 0
		} else {
			c.Idle++
		}
		if c.Idle >= 900 && c.onDemo != nil { // 0x42ce3e
			index := g.Selection.DemoIndex
			g.Selection.DemoIndex = (index + 1) & 3 // PE 0x476ff8: 1/2/3/0
			g.TitleEntry = 0
			if c.State != TitleIntro {
				g.TitleEntry = 1
			}
			c.Idle = 0
			if err := c.onDemo(g, index); err != nil {
				return err
			}
		}
	}
	switch c.State {
	case TitleInitialize:
		return c.initialize(g, in)
	case TitleIntro:
		c.calcIntro(g, m)
	case TitleMain:
		c.calcMain(g, m)
	case TitleQuit:
		g.RequestScene(game.SceneExit) // FUN_0042cdf0 state 3 -> FUN_0040ac90
	case TitleDifficulty:
		c.calcDifficulty(g, m)
	case TitleCharacter:
		c.calcCharacter(g, m)
	case TitleShot:
		return c.calcShot(g, m)
	case TitlePractice:
		return c.calcPractice(g, in)
	case TitleOption:
		return c.calcOption(g, m)
	case TitleKeys:
		c.calcKeys(g, in)
	}
	return nil
}

// initialize is FUN_0042cdf0 state 0, including its same-calc dispatch.
func (c *TitleController) initialize(g *game.Game, in *input.State) error {
	c.World.Delete(c.LoadingMarker)
	c.LoadingMarker = nil
	demoReturn := g.RunState != nil && g.RunState.RunFlags&game.RunFlagDemo != 0
	if demoReturn {
		c.Difficulty = g.Selection.Difficulty // DAT_00474c78 restores only difficulty
		g.RunState.RunFlags &^= game.RunFlagDemo
	} else if c.entry != 3 && g.Audio() != nil {
		g.Audio().PlayBGM(audio.BGMTitle)
	}
	switch c.entry {
	case 0:
		c.SetState(TitleIntro)
		g.TitleEntry = 1
		c.calcIntro(g, in.Menu())
	case 2, 3:
		c.Selector.Count = 10
		c.Selector.Set(3)
		page := TitleReplay
		if c.entry == 3 {
			c.Selector.Set(0)
			page = TitlePlayerDataDetail
		}
		c.Selector.Push()
		c.SetState(page)
		c.Spawn(0x5b)
		c.Interrupt(0x5b, 9)
		g.TitleEntry = 1
		if h := c.Hooks[page]; h.Calc != nil {
			return h.Calc(c, g, in)
		}
	default:
		if c.Difficulty == game.DiffExtra {
			c.Selector.Set(1)
		}
		c.SetState(TitleMain)
		c.Spawn(0x5b)
		c.calcMain(g, in.Menu())
	}
	return nil
}

// calcIntro is FUN_0042d300; state 1 is a real title, not an immediate menu.
func (c *TitleController) calcIntro(g *game.Game, m input.MenuInput) {
	switch c.Substate {
	case 0:
		if c.Root(0x58) == nil {
			c.Spawn(0x58)
			c.Spawn(0x5a)
			c.version = c.World.SpawnTail(c.VersionFile, 0, 15, c.Env)
		}
		c.SetSubstate(1)
		c.Selector.Set(0)
	case 1:
		if c.Timer.Cur > 10 {
			c.SetSubstate(2)
		}
	case 2:
		if m.Pressed&input.MenuIdleReset != 0 {
			c.Interrupt(0x5a, 6)
			c.SetSubstate(4)
			c.playSE(g, 0x20)
			c.World.Delete(c.version)
		}
	case 4:
		if c.Timer.Cur >= 30 {
			c.SetState(TitleMain)
		}
	}
}

// calcMain is FUN_0042d420. The eight-item root owns its own text and
// cursor animations; labels 3 and 7+cursor are not new/restarted scripts.
func (c *TitleController) calcMain(g *game.Game, m input.MenuInput) {
	switch c.Substate {
	case 0:
		c.Selector.Count = 8
		c.Selector.Disabled = nil
		available := false
		for shot := range 6 {
			available = available || ExtraAvailable(g.Save, shot)
		}
		if !available {
			c.Selector.Disabled = append(c.Selector.Disabled, 1)
		}
		if c.Practice {
			c.Selector.Set(2)
			c.Practice = false
			if g.RunState != nil {
				g.RunState.RunFlags &^= game.RunFlagPractice
			}
		}
		if c.Root(0x58) == nil {
			c.Spawn(0x58)
		}
		c.Spawn(0)
		c.SetSubstate(1)
	case 1:
		if c.Timer.Cur > 10 {
			c.SetSubstate(2)
			c.Interrupt(0, 3)
			c.Interrupt(0, int16(17+c.Selector.Value))
		}
	case 2:
		c.move(g, m, input.MenuUp, input.MenuDown, 0)
		if m.Pressed&input.MenuCancel != 0 {
			c.playSE(g, 11)
			if c.Selector.Value == 7 {
				c.SetSubstate(4)
				return
			}
			c.Selector.Set(7)
			c.Interrupt(0, 3)
			c.Interrupt(0, 14)
		}
		if m.Pressed&input.MenuConfirm != 0 {
			c.Interrupt(0, 6)
			c.SetSubstate(4)
			switch c.Selector.Value {
			case 7:
				c.playSE(g, 11)
			case 6:
				c.playSE(g, 10)
				c.Interrupt(0x5a, 3)
				c.Interrupt(0x5b, 3)
			default:
				c.playSE(g, 10)
				c.Interrupt(0x5a, 7)
				c.Interrupt(0x5b, 7)
				if c.Selector.Value == 5 {
					c.playSE(g, 10)
				} // 0x42d6ad
			}
		}
	case 4:
		if c.Timer.Cur < 20 {
			return
		}
		choice := c.Selector.Value
		if choice == 7 {
			c.SetState(TitleQuit)
			return
		}
		c.Selector.Push()
		if choice != 6 {
			c.LeaveScript(0x58)
		}
		switch choice {
		case 0, 1, 2:
			c.Practice = choice == 2
			if choice == 1 {
				c.oldDifficulty, c.Difficulty = c.Difficulty, game.DiffExtra
			} else if c.Difficulty > 3 {
				c.Difficulty = game.DiffNormal
			}
			c.SetState(TitleDifficulty)
			if choice == 1 {
				c.Selector.Set(0)
			} else {
				c.Selector.Set(c.Difficulty)
			}
		case 3:
			c.SetState(TitleReplay)
		case 4:
			c.SetState(TitlePlayerData)
		case 5:
			c.SetState(TitleMusicRoom)
		case 6:
			c.SetState(TitleOption)
		}
	}
}

func (c *TitleController) move(g *game.Game, m input.MenuInput, previous, next uint16, script int) {
	c.Selector.Previous = c.Selector.Value
	if m.Move(previous) {
		c.Selector.Move(-1)
	}
	if m.Move(next) {
		c.Selector.Move(1)
	}
	if c.Selector.Value != c.Selector.Previous {
		c.playSE(g, 12)
		c.Interrupt(script, 3)
		c.Interrupt(script, int16(7+c.Selector.Value))
	}
}

// calcDifficulty is FUN_00430320: up/down, then cancel, then confirm.
func (c *TitleController) calcDifficulty(g *game.Game, m input.MenuInput) {
	root := 0x77
	if c.Difficulty == game.DiffExtra {
		root = 0x78
	}
	switch c.Substate {
	case 0:
		if c.Root(0x5e) == nil {
			c.Spawn(0x5e)
			c.difficultyMarker = c.World.SpawnTail(titleASCII(), 8, 15, c.Env)
		}
		c.Selector.Count = 4
		if root == 0x78 {
			c.Selector.Count = 1
		}
		c.LeaveScript(root)
		c.Spawn(root)
		c.Interrupt(root, 3)
		c.Interrupt(root, int16(17+c.Selector.Value))
		c.Spawn(0x62)
		c.SetSubstate(1)
	case 1:
		if c.Timer.Cur > 6 {
			c.SetSubstate(2)
		}
	case 2:
		if root == 0x77 {
			c.move(g, m, input.MenuUp, input.MenuDown, root)
		}
		if m.Pressed&input.MenuCancel != 0 {
			c.SetSubstate(4)
			c.playSE(g, 11)
			c.LeaveScript(root)
			return
		}
		if m.Pressed&input.MenuConfirm != 0 {
			c.Interrupt(root, 6)
			child := 0x6d + c.Selector.Value
			if root == 0x78 {
				child = 0x71
			}
			c.InterruptChild(root, child, 2)
			c.SetSubstate(3)
			c.playSE(g, 10)
		}
	case 3:
		if c.Timer.Cur >= 14 {
			c.LeaveScript(0x62)
			if root == 0x77 {
				c.Difficulty = c.Selector.Value
			}
			c.SetState(TitleCharacter)
			c.Selector.Push()
			c.Selector.Count = 2
			c.Selector.Set(c.Character)
		}
	case 4:
		if c.Timer.Cur >= 6 {
			c.LeaveScript(0x62)
			c.LeaveScript(0x5e)
			c.Interrupt(0x5a, 8)
			c.Interrupt(0x5b, 8)
			c.World.Delete(c.difficultyMarker)
			c.difficultyMarker = nil
			if root == 0x78 {
				c.Difficulty = c.oldDifficulty
			} else {
				c.Difficulty = c.Selector.Value
			}
			c.SetState(TitleMain)
			c.Selector.Pop()
		}
	}
}

// calcCharacter is FUN_004306a0, with the two independent portraits.
func (c *TitleController) calcCharacter(g *game.Game, m input.MenuInput) {
	switch c.Substate {
	case 0:
		c.Selector.Count, c.Selector.Disabled = 2, nil
		if c.Difficulty == game.DiffExtra {
			for character := range 2 {
				available := false
				for shot := range 3 {
					available = available || ExtraAvailable(g.Save, character*3+shot)
				}
				if !available {
					c.Selector.Disabled = append(c.Selector.Disabled, character)
					if c.Selector.Value == character {
						c.Selector.Set(1 - character)
					}
				}
			}
		}
		c.Spawn(0x63)
		c.LeaveScript(0x7d)
		c.Spawn(0x7d)
		c.Interrupt(0x7d, 3)
		c.Interrupt(0x7d, int16(17+c.Selector.Value))
		c.SetSubstate(1)
	case 1:
		if c.Timer.Cur > 6 {
			c.SetSubstate(2)
		}
	case 2:
		c.move(g, m, input.MenuLeft, input.MenuRight, 0x7d)
		if m.Pressed&input.MenuCancel != 0 {
			c.SetSubstate(4)
			c.playSE(g, 11)
			return
		}
		if m.Pressed&input.MenuConfirm != 0 {
			n := c.Selector.Value
			c.InterruptChild(0x7d, 0x79+n, 6)
			c.InterruptChild(0x7d, 0x7b+n, 6)
			c.InterruptChild(0x7d, 0x7a-n, 1)
			c.InterruptChild(0x7d, 0x7c-n, 1)
			c.SetSubstate(3)
			c.playSE(g, 10)
		}
	case 3:
		if c.Timer.Cur >= 14 {
			c.LeaveScript(0x63)
			c.SetState(TitleShot)
			old := c.Character
			c.Character = c.Selector.Value
			c.Selector.Push()
			c.Selector.Count = 3
			if old != c.Character {
				c.Selector.Set(0)
			} else {
				c.Selector.Set(c.Shot)
			}
		}
	case 4:
		if c.Timer.Cur >= 6 {
			c.LeaveScript(0x7d)
			c.LeaveScript(0x63)
			c.Character = c.Selector.Value
			c.SetState(TitleDifficulty)
			c.Selector.Pop()
		}
	}
}

// calcShot is FUN_00430a60. Practice branches at exactly timer 10;
// a normal run creates the loading marker/fade at 10 and requests 7 at 40.
func (c *TitleController) calcShot(g *game.Game, m input.MenuInput) error {
	root := 0x96 + c.Character
	switch c.Substate {
	case 0:
		c.Selector.Count, c.Selector.Disabled = 3, nil
		if c.Difficulty == game.DiffExtra {
			for shot := range 3 {
				if !ExtraAvailable(g.Save, c.Character*3+shot) {
					c.Selector.Disabled = append(c.Selector.Disabled, shot)
				}
			}
			if !ExtraAvailable(g.Save, c.Character*3+c.Selector.Value) {
				for shot := range 3 {
					if ExtraAvailable(g.Save, c.Character*3+shot) {
						c.Selector.Set(shot)
						break
					}
				}
			}
		}
		c.Spawn(0x64)
		c.LeaveScript(root)
		c.Spawn(root)
		c.Interrupt(root, 3)
		c.Interrupt(root, int16(17+c.Selector.Value))
		c.SetSubstate(1)
		for shot := range 3 {
			if g.Save == nil || g.Save.Progress[c.Character*3+shot].Clears[c.Difficulty] == 0 {
				c.World.Hide(c.Child(root, 0x8a+c.Character*3+shot)) // 0x430c98..0x430d39
			}
		}
	case 1:
		if c.Timer.Cur > 6 {
			c.SetSubstate(2)
		}
	case 2:
		c.move(g, m, input.MenuUp, input.MenuDown, root)
		if m.Pressed&input.MenuCancel != 0 {
			c.SetSubstate(4)
			c.playSE(g, 11)
			return nil
		}
		if m.Pressed&input.MenuConfirm != 0 {
			c.SetSubstate(3)
			c.playSE(g, 10)
			c.InterruptChild(root, 0x7e+c.Character*6+c.Selector.Value, 6)
		}
	case 3:
		if c.Timer.Cur == 10 {
			if c.Practice {
				c.Shot = c.Selector.Value
				c.Selector.Push()
				c.LeaveScript(0x64)
				c.SetState(TitlePractice)
				return nil
			}
			c.ShowLoading()
			c.AddFade(5, 32, 0) // 0x430e8f..0x430eb3
		}
		if c.Timer.Cur >= 40 {
			c.Shot = c.Selector.Value
			c.Selector.Push()
			c.SetState(TitleQuit)
			return c.launch(g, 1)
		}
	case 4:
		if c.Timer.Cur >= 6 {
			c.LeaveScript(root)
			c.LeaveScript(0x64)
			c.SetState(TitleCharacter)
			c.Selector.Pop()
		}
	}
	return nil
}

func (c *TitleController) PracticeAvailable(stage int) bool {
	if stage < 1 || stage > 6 || c.Difficulty < 0 || c.Difficulty >= 4 || c.Game.Save == nil {
		return false
	}
	return c.Game.Save.Progress[c.Character*3+c.Shot].Stages[c.Difficulty][stage-1].Cleared != 0 // 0x43114f
}

// AvailableStages follows the task's filtered-display policy; the stored
// selector value remains the original zero-based stage index, not a row.
func (c *TitleController) AvailableStages() []int {
	var stages []int
	for stage := 1; stage <= 6; stage++ {
		if c.PracticeAvailable(stage) {
			stages = append(stages, stage)
		}
	}
	return stages
}

// calcPractice is FUN_00430ff0. Only cleared records may start a stage.
func (c *TitleController) calcPractice(g *game.Game, in *input.State) error {
	m := in.Menu()
	switch c.Substate {
	case 0:
		c.Selector.Count = 6
		c.Selector.Set(c.PracticeStage)
		c.Selector.Disabled = nil
		for stage := 1; stage <= 6; stage++ {
			if !c.PracticeAvailable(stage) {
				c.Selector.Disabled = append(c.Selector.Disabled, stage-1)
			}
		}
		if !c.PracticeAvailable(c.Selector.Value + 1) {
			if stages := c.AvailableStages(); len(stages) != 0 {
				c.Selector.Set(stages[0] - 1)
			}
		}
		c.Spawn(0x6a)
		c.Spawn(0x6b + c.Character)
		c.SetSubstate(1)
	case 1:
		if c.Timer.Cur > 10 {
			c.SetSubstate(2)
		}
	case 2:
		c.Selector.Previous = c.Selector.Value
		if m.Move(input.MenuUp) {
			c.Selector.Move(-1)
		}
		if m.Move(input.MenuDown) {
			c.Selector.Move(1)
		}
		if c.Selector.Value != c.Selector.Previous {
			c.playSE(g, 12)
		}
		if m.Pressed&input.MenuCancel != 0 {
			c.PracticeStage = c.Selector.Value
			c.SetSubstate(4)
			c.playSE(g, 11)
			return nil
		}
		if m.Pressed&input.MenuConfirm != 0 {
			c.PracticeStage, c.PracticeLives = c.Selector.Value, in.PracticeLives()
			if !c.PracticeAvailable(c.Selector.Value + 1) {
				c.playSE(g, 0x25)
				return nil
			}
			c.SetSubstate(3)
			c.playSE(g, 10)
		}
	case 3:
		if c.Timer.Cur == 10 {
			c.ShowLoading()
			c.AddFade(5, 32, 0)
		} // 0x431320..0x43134d
		if c.Timer.Cur >= 40 {
			stage := c.Selector.Value + 1
			c.Selector.Push()
			c.SetState(TitleQuit)
			return c.launch(g, stage)
		}
	case 4:
		if c.Timer.Cur >= 6 {
			c.LeaveScript(0x6a)
			c.LeaveScript(0x6b + c.Character)
			c.SetState(TitleShot)
			c.Selector.Pop()
		}
	}
	return nil
}

// launch writes t8's native mode/resource fields; no new practice flags or
// alternate replay stream is invented in the frontend.
func (c *TitleController) launch(g *game.Game, stage int) error {
	s := g.NewState(c.Character, c.Shot, c.Difficulty)
	if c.Difficulty == game.DiffExtra {
		stage = game.StageExtra
	}
	s.Stage, s.StartStage, s.StageMode, s.ReplayMode, s.FreshRun = stage, stage, 0, 0, true
	if stage != 1 {
		s.Power = 0x50 * 5
	} // FUN_00417870, 0x417976..0x41798c, including Extra
	if c.Practice {
		s.PracticeLives = c.PracticeLives
		s.Practice(stage)
	}
	if g.Audio() != nil {
		g.Audio().FadeOutBGM(360)
	} // FUN_00420c30: PE 0x40c00000 seconds
	if c.onStart == nil {
		return fmt.Errorf("title: stage constructor is not registered")
	}
	return c.onStart(g, s)
}

// GlyphText is FUN_00401760's native small-font path, shared with t11.
func GlyphText(sv *supervisor.Supervisor, text string, x, y float32, colour uint32) {
	GlyphTextFile(sv, titleASCII(), text, x, y, colour)
}

// drawPractice is FUN_00431410, with only unlocked rows per the task.
func (c *TitleController) drawPractice(sv *supervisor.Supervisor) {
	if c.Substate < 2 || c.Substate > 3 || (c.Timer.Cur < 10 && c.Substate != 3) {
		return
	}
	x := float32(168)
	if c.Character != 0 {
		x = 296
	} // 0x431465..0x43147e
	for _, stage := range c.AvailableStages() {
		colour := uint32(0xff808080)
		if c.Selector.Value == stage-1 {
			colour = 0xffffff00
			if c.Substate == 3 && c.Timer.Cur%4 >= 2 {
				colour = 0xff000000
			}
		}
		r := c.Game.Save.Progress[c.Character*3+c.Shot].Stages[c.Difficulty][stage-1]
		GlyphText(sv, fmt.Sprintf("Stage %d  %08d0", stage, r.Score), x, 216+float32(stage-1)*math.Float32frombits(0x41900000), colour) // PE 0x46effc
	}
}
