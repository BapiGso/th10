package ecl

import (
	"fmt"
	"strings"
)

// Disassemble renders the whole program in a thtk-like text form. This exists
// purely for debugging — the binary IR is the source of truth, and this lets a
// human eyeball what was decoded (the one thing the text format gave for free).
func (p *Program) Disassemble() string {
	var b strings.Builder
	if len(p.AnimNames) > 0 {
		fmt.Fprintf(&b, "anim { %s }\n", quoteList(p.AnimNames))
	}
	if len(p.EcliNames) > 0 {
		fmt.Fprintf(&b, "ecli { %s }\n", quoteList(p.EcliNames))
	}
	for _, sub := range p.Subs {
		b.WriteString("\n")
		b.WriteString(sub.Disassemble())
	}
	return b.String()
}

// Disassemble renders a single sub.
func (s *Sub) Disassemble() string {
	var b strings.Builder
	fmt.Fprintf(&b, "void %s() {\n", s.Name)
	lastTime := int32(0)
	lastRank := uint8(0xff)
	for _, in := range s.Instrs {
		if in.Time != lastTime {
			fmt.Fprintf(&b, "  +%d: // %d\n", in.Time-lastTime, in.Time)
			lastTime = in.Time
		}
		if in.RankMask != lastRank {
			fmt.Fprintf(&b, "  !%s\n", rankString(in.RankMask))
			lastRank = in.RankMask
		}
		fmt.Fprintf(&b, "    %s(%s); // @%d\n", opName(in.Opcode), in.argString(), in.Offset)
	}
	b.WriteString("}\n")
	return b.String()
}

func (in Instr) argString() string {
	parts := make([]string, 0, len(in.Params))
	for _, p := range in.Params {
		parts = append(parts, p.String(in.Offset))
	}
	return strings.Join(parts, ", ")
}

// String renders one decoded parameter; instrOff is needed to resolve 'o' jumps.
func (p Param) String(instrOff int32) string {
	if p.IsVar {
		id := p.VarID()
		sig := byte('$')
		if p.Type == 'f' {
			sig = '%'
		}
		switch {
		case id == -1:
			return "[pop]"
		case id < -1:
			return fmt.Sprintf("[gvar%d]", id)
		default:
			// local frame variable: byte offset, A=offset 0, B=4, ...
			return fmt.Sprintf("%c%s", sig, varName(id))
		}
	}
	switch p.Type {
	case 'f':
		return fmt.Sprintf("%gf", p.Float())
	case 'm', 'x':
		return fmt.Sprintf("%q", p.Str)
	case 'o':
		return fmt.Sprintf("@%d", instrOff+p.Int())
	case 't':
		return fmt.Sprintf("t%d", p.Int())
	case 'D':
		if p.DFrom == 'f' {
			return fmt.Sprintf("%gf", p.Float())
		}
		return fmt.Sprintf("%d", p.Int())
	default:
		return fmt.Sprintf("%d", p.Int())
	}
}

func quoteList(ss []string) string {
	parts := make([]string, len(ss))
	for i, s := range ss {
		parts[i] = fmt.Sprintf("%q;", s)
	}
	return strings.Join(parts, " ")
}

// varName renders a local-frame byte offset as a thtk-style bijective base-26
// name: offset 0 -> A, 4 -> B, 8 -> C, 12 -> D, ...
func varName(byteOffset int32) string {
	n := int(byteOffset)/4 + 1
	var b []byte
	for n > 0 {
		n--
		b = append([]byte{byte('A' + n%26)}, b...)
		n /= 26
	}
	return string(b)
}

func rankString(mask uint8) string {
	if mask == 0xff {
		return "*"
	}
	var b strings.Builder
	for i, ch := range []byte{'E', 'N', 'H', 'L'} {
		if mask&(1<<uint(i)) != 0 {
			b.WriteByte(ch)
		}
	}
	return b.String()
}

func opName(op uint16) string {
	if name, ok := opNames[op]; ok {
		return name
	}
	return fmt.Sprintf("ins_%d", op)
}

// opNames covers the opcodes we actually interpret; everything else prints ins_N.
var opNames = map[uint16]string{
	opRetBig: "delete", opRetNormal: "return", opCall: "call", opGoto: "goto",
	opUnless: "unless", opIf: "if", opCallAsync: "callAsync", opCallAsyncID: "callAsyncId",
	opStackAlloc: "var", opLoadI: "pushI", opAssignI: "setI", opLoadF: "pushF", opAssignF: "setF",
	opWait: "wait", opPolar: "polar", opValidRad: "validRad", opDec: "dec",
	opEnmCreate: "enmCreate", opAnmSelect: "anmSelect", opAnmSetMain: "anmSetMain",
	opMovePosTime: "movePosTime", opSetVel: "setVel", opMoveVelTime: "moveVelTime",
	opMoveRand: "moveRand", opMoveLimit: "moveLimit", opSetHP: "setHP", opSetBoss: "setBoss",
	opETNew: "etNew", opETFire: "etFire", opETSprite: "etSprite", opETAngle: "etAngle",
	opETSpeed: "etSpeed", opETCount: "etCount", opETAim: "etAim", opETEx: "etEx",
	opLaserOnA: "laserOnA", opLaserStOn: "laserStOn", opETCancel: "etCancel", opETClear: "etClear",
}
