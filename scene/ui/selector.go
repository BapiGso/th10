package ui

// Selector is FUN_0044bea0's cursor, wrap policy and disabled-index list.
// Push/Pop are FUN_0044be20/0044be70, including their depth-15 clamp.
type Selector struct {
	Value, Previous, Count int
	Wrap                   bool
	Disabled               []int
	stack                  [16]struct{ value, count int }
	depth                  int
}

// Set is FUN_0040ad20: a zero item count permits the supplied value.
func (s *Selector) Set(value int) {
	if s.Count != 0 {
		value = min(max(value, 0), s.Count-1)
	}
	s.Value = value
}

func (s *Selector) Move(delta int) int {
	if s.Count <= 0 {
		return s.Value
	}
	s.Value += delta
	for tries := 0; tries <= s.Count; tries++ {
		if s.Wrap {
			s.Value = ((s.Value % s.Count) + s.Count) % s.Count
		} else {
			s.Value = min(max(s.Value, 0), s.Count-1)
		}
		disabled := false
		for _, value := range s.Disabled {
			disabled = disabled || value == s.Value
		}
		if !disabled {
			return s.Value
		}
		s.Value += delta // 0x44bf0f jumps back before the delta addition
	}
	// Native callers never disable every item. Keep malformed host data
	// from hanging the calc chain instead of inventing a selectable item.
	s.Value = s.Previous
	return s.Value
}

func (s *Selector) Push() {
	s.stack[s.depth].value, s.stack[s.depth].count = s.Value, s.Count
	s.depth = min(s.depth+1, 15) // 0x44be42..0x44be57
	s.Disabled = nil
}

func (s *Selector) Pop() {
	s.depth = max(s.depth-1, 0) // 0x44be70..0x44be83
	s.Value, s.Count = s.stack[s.depth].value, s.stack[s.depth].count
	s.Disabled = nil
}
