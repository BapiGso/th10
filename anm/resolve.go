package anm

// ANM script opcodes we interpret for sprite resolution. The full instruction
// set (~75 ops: color/scale/rotation/blend/interpolation) is handled by the
// animation VM in track B2; here we only need the sprite-timeline subset.
const (
	opEnd       = 0 // ins_0: end of script
	opStop      = 1 // ins_1: stop/return (terminal, like end for our purposes)
	opSetSprite = 3 // ins_3(spriteID): show this sprite
	opGoto      = 4 // ins_4(byteOffset, time): jump within the script, set clock=time
	opLoop      = 5 // ins_5(count, byteOffset, time): decrement counter, jump while >0
)

// SpriteAt walks a script's ins_3/ins_4/ins_5 timeline and returns the sprite
// id displayed at the given age (in frames), or -1 if the script shows none.
//
// This is the faithful clock model (same as the ECL VM): an instruction runs
// once its Time <= clock; the clock advances one per frame; ins_4/ins_5 reset
// the clock to their time arg. A step budget bounds infinite idle loops.
func (s *Script) SpriteAt(age int) int {
	if s == nil || len(s.Instrs) == 0 {
		return -1
	}
	offsetIndex := s.offsetIndex()
	sprite := -1
	clock := 0   // virtual clock; loops/resets on goto
	elapsed := 0 // real frames since start; monotonic (termination)
	pc := 0
	loopCounters := map[int]int{}
	for budget := 0; budget < 1_000_000; budget++ {
		if pc < 0 || pc >= len(s.Instrs) {
			break
		}
		in := s.Instrs[pc]
		if in.Time > clock {
			if elapsed >= age {
				break // reached the queried frame
			}
			clock++
			elapsed++
			continue
		}
		switch in.Op {
		case opEnd, opStop:
			return sprite
		case opSetSprite:
			sprite = int(in.Arg(0))
			pc++
		case opGoto:
			target, ok := offsetIndex[int(in.Arg(0))]
			if !ok {
				return sprite
			}
			pc = target
			clock = int(in.Arg(1))
		case opLoop:
			n := loopCounters[in.Offset]
			if n == 0 {
				n = int(in.Arg(0))
			}
			n--
			if n > 0 {
				loopCounters[in.Offset] = n
				target, ok := offsetIndex[int(in.Arg(1))]
				if !ok {
					return sprite
				}
				pc = target
				clock = int(in.Arg(2))
			} else {
				loopCounters[in.Offset] = 0
				pc++
			}
		default:
			pc++
		}
	}
	return sprite
}

func (s *Script) offsetIndex() map[int]int {
	m := make(map[int]int, len(s.Instrs))
	for i, in := range s.Instrs {
		m[in.Offset] = i
	}
	return m
}
