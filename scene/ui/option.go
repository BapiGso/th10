package ui

import (
	"th10/game"
	"th10/input"
)

func optionConfig(g *game.Game) *game.NativeConfig {
	if g.Config == nil {
		g.Config = game.DefaultConfig()
	}
	if g.Config.Native == nil {
		g.Config.Native = game.DefaultNativeConfig()
	}
	bytes := g.Config.NativeBytes()
	g.Config.Native.Raw[0x20], g.Config.Native.Raw[0x21] = bytes[0x20], bytes[0x21]
	return g.Config.Native
}

// refreshHint is FUN_0042d920's 0x42d9aa/0x42dd8b/0x42e044 branches:
// set sprites, never restart the Hint children with different script ids.
func (c *TitleController) refreshHint(g *game.Game) {
	native := optionConfig(g)
	pair := [2]int{0x2e, 0x2f} // On
	switch native.Raw[0x22] {
	case 1:
		pair = [2]int{0x30, 0x31}
	case 2:
		pair = [2]int{0x2c, 0x2d}
	}
	c.SetChildSprite(1, 0x1f, pair[0])
	c.SetChildSprite(1, 0x20, pair[1])
	g.Config.Hint = native.Raw[0x22] != 1 // preserve the host bool without losing native Auto
}

// refreshVolumes is FUN_0042e5a0: six digits plus their six shadows,
// including both leading-digit visibility flags.
func (c *TitleController) refreshVolumes(g *game.Game) {
	native := optionConfig(g)
	for _, group := range [...]struct {
		value         byte
		first, shadow int
	}{
		{native.Raw[0x20], 0x25, 0x29}, {native.Raw[0x21], 0x2d, 0x31},
	} {
		n := int(group.value)
		for digit, value := range [...]int{n / 100, n / 10 % 10, n % 10} {
			c.SetChildSprite(1, group.first+digit, 0x33+value)
			c.SetChildSprite(1, group.shadow+digit, 0x3d+value)
			show := digit == 2 || (digit == 1 && n >= 10) || n >= 100
			for _, script := range [...]int{group.first + digit, group.shadow + digit} {
				vm := c.Child(1, script)
				if show {
					c.World.Show(vm)
				} else {
					c.World.Hide(vm)
				}
			}
		}
	}
	if g.Audio() != nil {
		g.Audio().SetNativeVolumes(native.Raw[0x20], native.Raw[0x21])
	}
}

// calcOption is FUN_0042d920, six native items, no ebitenui sliders.
func (c *TitleController) calcOption(g *game.Game, m input.MenuInput) error {
	native := optionConfig(g)
	switch c.Substate {
	case 0:
		c.Selector.Count = 6
		c.Selector.Set(0)
		c.Spawn(1)
		c.refreshVolumes(g)
		c.SetSubstate(1)
	case 1:
		if c.Timer.Cur > 6 {
			c.SetSubstate(2)
			c.Interrupt(1, 3)
			c.refreshHint(g)
			c.Interrupt(1, int16(17+c.Selector.Value))
		}
	case 2:
		c.move(g, m, input.MenuUp, input.MenuDown, 1)
		if m.Pressed&input.MenuCancel != 0 {
			c.playSE(g, 11)
			if c.Selector.Value == 5 {
				c.Interrupt(1, 6)
				c.SetSubstate(4)
			} else {
				c.Selector.Set(5)
				c.Interrupt(1, 3)
				c.Interrupt(1, 12)
			}
			return nil // 0x42dcc8 returns before left/right/confirm
		}
		if c.Selector.Value == 2 && c.Timer.Cur%60 == 0 {
			c.playSE(g, 4)
		} // 0x42dccb
		for _, direction := range [...]struct {
			mask  uint16
			delta int
		}{{input.MenuLeft, -1}, {input.MenuRight, 1}} {
			if !m.Move(direction.mask) {
				continue
			}
			switch c.Selector.Value {
			case 0:
				native.Raw[0x22] = byte((int(native.Raw[0x22]) + direction.delta + 3) % 3)
				c.refreshHint(g)
				c.playSE(g, 12)
			case 1:
				g.Config.BGMVolume = float64(min(max(int(native.Raw[0x20])+5*direction.delta, 0), 100)) / 100
				c.refreshVolumes(g)
			case 2:
				g.Config.SEVolume = float64(min(max(int(native.Raw[0x21])+5*direction.delta, 0), 100)) / 100
				c.refreshVolumes(g)
			}
		}
		if m.Pressed&input.MenuConfirm != 0 {
			switch c.Selector.Value {
			case 3:
				c.Interrupt(1, 6)
				c.playSE(g, 10)
				c.SetSubstate(4)
			case 4:
				g.Config.BGMVolume, g.Config.SEVolume, native.Raw[0x22] = 1, 0.8, 0 // 0x42e2ce
				c.refreshVolumes(g)
				c.refreshHint(g)
				c.playSE(g, 10)
			case 5:
				c.Interrupt(1, 6)
				c.playSE(g, 11)
				c.SetSubstate(4)
			}
		}
	case 4:
		if c.Timer.Cur >= 10 {
			if c.Selector.Value == 3 {
				c.SetState(TitleKeys)
				c.Selector.Push()
			} else {
				c.Interrupt(0x5a, 2)
				c.Interrupt(0x5b, 2)
				c.SetState(TitleMain)
				c.Selector.Pop()
				return g.SaveConfig() // host persistence boundary; raw cfg bytes are preserved
			}
		}
	}
	return nil
}

func (c *TitleController) currentKeys(g *game.Game) [5]int16 {
	keys := optionConfig(g).JoystickBindings()
	return [5]int16{keys[0], keys[1], keys[2], keys[3], keys[8]}
}

// refreshKeys is FUN_0042f8b0: signed division/remainder, including -1
// for an unbound button. The shadow scripts use the same digit sprite ids.
func (c *TitleController) refreshKeys() {
	for key, value := range c.keyBindings {
		for _, first := range [...]int{0x43 + key*2, 0x4d + key*2} {
			c.SetChildSprite(2, first, 0x33+int(value)/10)
			c.SetChildSprite(2, first+1, 0x33+int(value)%10)
		}
	}
}

// BindButton is FUN_00430250: a duplicate swaps with the selected key's
// old button instead of clearing the other binding or allowing duplicates.
func (c *TitleController) BindButton(key, button int) bool {
	if key < 0 || key >= 5 || button < 0 || button >= 31 || c.keyBindings[key] == int16(button) {
		return false
	}
	old := c.keyBindings[key]
	for i := range c.keyBindings {
		if i != key && c.keyBindings[i] == int16(button) {
			c.keyBindings[i] = old
		}
	}
	c.keyBindings[key] = int16(button)
	c.refreshKeys()
	return true
}

// calcKeys is FUN_0042f540: five joystick keys, restore-current, Quit.
// Only confirming Quit commits; cancel on Quit restores the live table.
func (c *TitleController) calcKeys(g *game.Game, in *input.State) {
	m := in.Menu()
	switch c.Substate {
	case 0:
		c.Selector.Count = 7
		c.Selector.Set(0)
		c.Spawn(2)
		c.keyBindings = c.currentKeys(g)
		c.refreshKeys()
		c.SetSubstate(1)
	case 1:
		if c.Timer.Cur > 6 {
			c.SetSubstate(2)
			c.Interrupt(2, 3)
			c.Interrupt(2, int16(17+c.Selector.Value))
		}
	case 2:
		c.move(g, m, input.MenuUp, input.MenuDown, 2)
		if c.Selector.Value < 5 {
			for button := 0; button < 31; button++ {
				if in.MenuButtons()&(1<<button) != 0 {
					if c.BindButton(c.Selector.Value, button) {
						c.playSE(g, 10)
					}
					break
				}
			}
		}
		if m.Pressed&input.MenuCancel != 0 && c.Selector.Value == 6 {
			c.keyBindings = c.currentKeys(g)
			c.refreshKeys()
			c.Interrupt(2, 6)
			c.playSE(g, 11)
			c.SetSubstate(4)
			return
		}
		if m.Pressed&input.MenuConfirm != 0 {
			switch c.Selector.Value {
			case 5:
				c.keyBindings = c.currentKeys(g)
				c.refreshKeys()
				c.playSE(g, 10) // 0x42f810 copies current live globals
			case 6:
				keys := optionConfig(g).JoystickBindings()
				for i, at := range [...]int{0, 1, 2, 3, 8} {
					keys[at] = c.keyBindings[i]
				}
				g.Config.Native.SetJoystickBindings(keys)
				in.SetGamepadBindings(keys)
				c.Interrupt(2, 6)
				c.playSE(g, 11)
				c.SetSubstate(4)
			}
		}
	case 4:
		if c.Timer.Cur >= 10 {
			c.SetState(TitleOption)
			c.Selector.Pop()
		}
	}
}
