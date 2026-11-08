package input

import "github.com/hajimehoshi/ebiten/v2"

// Native menu bits from FUN_0044a5f0. ReplayBits remains a different,
// direction-resolved representation and is never used as this bitset.
const (
	MenuShot        uint16 = 0x0001
	MenuBomb        uint16 = 0x0002
	MenuFocus       uint16 = 0x0004
	MenuPause       uint16 = 0x0008
	MenuUp          uint16 = 0x0010
	MenuDown        uint16 = 0x0020
	MenuLeft        uint16 = 0x0040
	MenuRight       uint16 = 0x0080
	MenuSkip        uint16 = 0x0100
	MenuReturnTitle uint16 = 0x0200
	MenuSpecial     uint16 = 0x0400
	MenuStart       uint16 = 0x0800
	MenuEnter       uint16 = 0x1000
	MenuRetry       uint16 = 0x4000
	MenuConfirm     uint16 = 0x1001
	MenuCancel      uint16 = 0x000a
	MenuIdleReset   uint16 = 0x160b // FUN_0042cdf0: directions/focus do not reset title idle
)

// MenuInput is FUN_0044a950's DAT_00474e30 snapshot. Update is the only
// operation that advances the per-bit counters; querying it is read-only.
type MenuInput struct {
	Current, Previous         uint16     // +0/+2
	Repeat, Pressed, Released uint16     // +4/+6/+8
	hold                      [16]uint16 // +0x0a
}

func (m *MenuInput) Update(raw uint16) {
	m.Previous, m.Current, m.Repeat = m.Current, raw, 0
	for bit := range m.hold {
		mask := uint16(1) << bit
		if raw&mask == 0 {
			m.hold[bit] = 0 // 0x44a99b
		} else {
			m.hold[bit]++
			if m.hold[bit] >= 26 { // 0x44a98a..0x44a994
				m.Repeat |= mask
				m.hold[bit] -= 8
			}
		}
	}
	m.Pressed, m.Released = raw&^m.Previous, m.Previous&^raw // 0x44a9ad..0x44a9c5
}

// Move is FUN_0040ace0. Confirm/cancel callers must use Pressed, not Move.
func (m MenuInput) Move(mask uint16) bool { return (m.Pressed|m.Repeat)&mask != 0 }
func (s *State) Menu() MenuInput          { return s.menu }

var menuKeyPairs = [...]struct {
	key Key
	bit uint16
}{
	{KeyShot, MenuShot}, {KeyBomb, MenuBomb}, {KeyFocus, MenuFocus}, {KeyPause, MenuPause},
	{KeyUp, MenuUp}, {KeyDown, MenuDown}, {KeyLeft, MenuLeft}, {KeyRight, MenuRight},
	{KeySkip, MenuSkip}, {KeyEnter, MenuEnter}, {KeyRetry, MenuRetry},
	{KeyReturnTitle, MenuReturnTitle}, {KeyStart, MenuStart}, {KeySpecial, MenuSpecial},
}

func menuKeys(keys [keyCount]bool) uint16 {
	var raw uint16
	for _, pair := range menuKeyPairs {
		if keys[pair.key] {
			raw |= pair.bit
		}
	}
	// Preserve the host's existing synthetic KeyOk/KeyCancel snapshots.
	// Actual keyboard sampling retains the distinct Enter/Z and Esc/X bits.
	if keys[KeyOk] && raw&MenuConfirm == 0 {
		raw |= MenuShot
	}
	if keys[KeyCancel] && raw&MenuCancel == 0 {
		raw |= MenuBomb
	}
	return raw
}

// RestoreMenu supplies the native, non-direction-resolved words to a
// headless menu. It does not touch the replay record's Aux1/Aux2 format.
func (s *State) RestoreMenu(raw uint16) {
	previous := s.menu
	var bits uint16
	for _, pair := range menuKeyPairs {
		if raw&pair.bit != 0 {
			bits |= 1 << pair.key
		}
	}
	if raw&MenuConfirm != 0 {
		bits |= 1 << KeyOk
	}
	if raw&MenuCancel != 0 {
		bits |= 1 << KeyCancel
	}
	s.Restore(bits)
	s.menu = previous
	s.menu.Update(raw) // retain all sixteen raw bits, even ones without a host Key
}

// SetMenuAux supplies the non-recorded joystick/digit sample in tests or
// another host. Digits are read on practice confirmation, not persisted cfg.
func (s *State) SetMenuAux(buttons uint32, lives int) {
	s.padButtons, s.practiceLives = buttons, min(max(lives, 0), 9)
}
func (s *State) MenuButtons() uint32 { return s.padButtons }
func (s *State) PracticeLives() int  { return s.practiceLives }

// SetGamepadBindings is cfg +4..+0x14, FUN_0044a1a0's five active words.
func (s *State) SetGamepadBindings(bindings [9]int16) {
	s.padBindings, s.padConfigured = bindings, true
}

func (s *State) sampleGamepad() {
	s.padButtons = 0
	ids := ebiten.AppendGamepadIDs(nil)
	if len(ids) == 0 {
		return
	}
	id := ids[0] // FUN_0044a5f0 samples joystick 0 for the primary menu/player
	for button := 0; button < min(ebiten.GamepadButtonCount(id), 32); button++ {
		if ebiten.IsGamepadButtonPressed(id, ebiten.GamepadButton(button)) {
			s.padButtons |= 1 << button
		}
	}
	bindings := s.padBindings
	if !s.padConfigured {
		bindings = [9]int16{0, 1, 2, 3, -1, -1, -1, -1, 4}
	}
	for _, p := range [...]struct {
		index int
		key   Key
	}{{0, KeyShot}, {1, KeyBomb}, {2, KeyFocus}, {3, KeyPause}, {8, KeySkip}} {
		button := bindings[p.index]
		if button >= 0 && button < 32 && s.padButtons&(1<<button) != 0 {
			s.pressed[p.key] = true
		}
	}
	if ebiten.GamepadAxisCount(id) >= 2 {
		x, y := ebiten.GamepadAxisValue(id, 0), ebiten.GamepadAxisValue(id, 1)
		// FUN_0044a1a0 uses quarter-range thresholds, strictly outside.
		s.pressed[KeyLeft] = s.pressed[KeyLeft] || x < -0.5
		s.pressed[KeyRight] = s.pressed[KeyRight] || x > 0.5
		s.pressed[KeyUp] = s.pressed[KeyUp] || y < -0.5
		s.pressed[KeyDown] = s.pressed[KeyDown] || y > 0.5
	}
	s.pressed[KeyOk] = s.pressed[KeyOk] || s.pressed[KeyShot]
	s.pressed[KeyCancel] = s.pressed[KeyCancel] || s.pressed[KeyBomb] || s.pressed[KeyPause]
}
