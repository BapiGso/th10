package dialog

// TH10 stage dialog (.msg) loader: the raw stXX_0C.msg th10.exe loads (C =
// character: 00 = Reimu, 01 = Marisa), decoded into the instructions
// FUN_00415e90 runs (msgvm.go).
//
// On-disk layout (little-endian):
//
//	u32 entry_count
//	entry_count × { u32 offset; u32 id }   (offset = byte offset of the stream)
//	per entry @offset: a stream of instructions until the next entry / EOF
//	instr: u16 time; u8 opcode; u8 length; [length bytes of args]
//
// Text arguments are XOR-encrypted Shift-JIS (thtk util_xor 0x77, 7, 0x10).

import (
	"encoding/binary"
	"fmt"
	"strings"
	"sync"

	"th10/gamedata"

	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/encoding/simplifiedchinese"
)

// ParseMSG decodes a TH10 .msg byte slice into its entries in file order, the
// order ECL 338 selects them by.
func ParseMSG(data []byte) [][]Instr {
	if len(data) < 4 {
		return nil
	}
	count := int(binary.LittleEndian.Uint32(data))
	if count <= 0 || 4+count*8 > len(data) {
		return nil
	}
	offsets := make([]int, count)
	for i := 0; i < count; i++ {
		offsets[i] = int(binary.LittleEndian.Uint32(data[4+i*8:]))
	}

	dec := japanese.ShiftJIS.NewDecoder()
	entries := make([][]Instr, count)
	for i, start := range offsets {
		end := len(data)
		if i+1 < count {
			end = offsets[i+1]
		}
		entries[i] = parseMSGStream(data, start, end, dec)
	}
	return entries
}

func parseMSGStream(data []byte, p, end int, dec interface {
	Bytes([]byte) ([]byte, error)
}) []Instr {
	var out []Instr
	for p+4 <= end {
		in := Instr{Time: binary.LittleEndian.Uint16(data[p:]), Op: data[p+2]}
		length := int(data[p+3])
		argOff := p + 4
		if argOff+length > end {
			break
		}
		args := data[argOff : argOff+length]
		switch in.Op {
		case msgLine1, msgLine2, msgText:
			in.Text = decodeMSGText(args, dec)
		default:
			if len(args) >= 4 {
				in.Arg = int32(binary.LittleEndian.Uint32(args))
			} else if len(args) > 0 {
				in.Arg = int32(args[0])
			}
		}
		out = append(out, in)
		p = argOff + length
	}
	return out
}

// decodeMSGText reverses thtk's util_xor(data, len, 0x77, 7, 0x10) then trims
// trailing NULs and decodes Shift-JIS → UTF-8.
func decodeMSGText(enc []byte, dec interface {
	Bytes([]byte) ([]byte, error)
}) string {
	buf := make([]byte, len(enc))
	key, step := 0x77, 7
	for i, b := range enc {
		buf[i] = b ^ byte(key)
		key = (key + step) & 0xff
		step = (step + 0x10) & 0xff
	}
	// strip trailing NUL padding
	n := len(buf)
	for n > 0 && buf[n-1] == 0 {
		n--
	}
	if n == 0 {
		return ""
	}
	if utf, err := dec.Bytes(buf[:n]); err == nil {
		return string(utf)
	}
	return ""
}

// ---------------------------------------------------------------------------
// Stage-level loader: official .msg, cached, with our speaker names.
// ---------------------------------------------------------------------------

// stageMsgSpeakers names the right (boss/enemy) speaker of each .msg entry of
// a stage, so the overlay can pick our portraits. Index = entry order in the
// file.
type stageMsgSpeakers struct {
	// rightBySeg[i] = boss/enemy speaker name for entry i (must exist in
	// speakerFaces).
	rightBySeg []string
}

// msgSpeakers maps stageN → per-entry right-speaker names. Entry layout (from
// the decoded files): most stages = [bossPre, post]; stage 3 = [midPre,
// midPost, bossPre, post]; extra = [midPre, bossPre, post]. Stage 1's
// mid-boss (静葉) has no dialog; stage 6's post entry has no text.
var msgSpeakers = map[string]stageMsgSpeakers{
	"stage1": {rightBySeg: []string{"秋穣子", "秋穣子"}},
	"stage2": {rightBySeg: []string{"鍵山雛", "鍵山雛"}},
	"stage3": {rightBySeg: []string{"河城にとり", "河城にとり", "河城にとり", "河城にとり"}},
	"stage4": {rightBySeg: []string{"射命丸文", "射命丸文"}},
	"stage5": {rightBySeg: []string{"東風谷早苗", "東風谷早苗"}},
	"stage6": {rightBySeg: []string{"八坂神奈子", "八坂神奈子"}},
	"extra":  {rightBySeg: []string{"八坂神奈子", "洩矢諏訪子", "洩矢諏訪子"}},
}

// StageMSG is a stage's .msg for one character.
type StageMSG struct {
	Entries    [][]Instr
	PlayerName string
	BossNames  []string // the right speaker of each entry
	Stage      int
}

var (
	msgCache   = map[string]*StageMSG{}
	msgCacheMu sync.Mutex
)

// LoadStage parses and caches stXX_0C.msg for a stage ("stage2", "extra")
// and character (0=Reimu, 1=Marisa). Nil when the file is missing or
// unparsable.
func LoadStage(stage string, character int) *StageMSG {
	stageNum := stageIndex(stage)
	if stageNum == 0 {
		return nil
	}
	key := fmt.Sprintf("%s_%d", stage, character&1)
	msgCacheMu.Lock()
	defer msgCacheMu.Unlock()
	if sm, ok := msgCache[key]; ok {
		return sm
	}

	var sm *StageMSG
	path := fmt.Sprintf("msg/st%02d_%02d.msg", stageNum, character&1)
	if data, err := gamedata.ReadFile(path); err == nil {
		if entries := ParseMSG(data); len(entries) > 0 {
			sm = &StageMSG{Entries: entries, PlayerName: "博麗霊夢", BossNames: msgSpeakers[stage].rightBySeg, Stage: stageNum}
			if character&1 == 1 {
				sm.PlayerName = "霧雨魔理沙"
			}
			if localized, err := gamedata.ReadFile(fmt.Sprintf("msg/st%02d_%02d.mst", stageNum, character&1)); err == nil {
				attachDisplayTexts(entries, localized)
			}
		}
	}
	msgCache[key] = sm
	return sm
}

// attachDisplayTexts extracts only th10c.dat's GBK-encoded traditional text.
// It never replaces a timestamp, opcode or argument with the patch's stream.
// Existing View/timing tests intentionally continue to see original text.
func attachDisplayTexts(entries [][]Instr, data []byte) {
	if len(data) < 4 {
		return
	}
	n := int(binary.LittleEndian.Uint32(data))
	if n != len(entries) || 4+n*8 > len(data) {
		return
	}
	dec := simplifiedchinese.GBK.NewDecoder()
	for i, code := range entries {
		start := int(binary.LittleEndian.Uint32(data[4+i*8:]))
		end := len(data)
		if i+1 < n {
			end = int(binary.LittleEndian.Uint32(data[4+(i+1)*8:]))
		}
		if start < 0 || start > end || end > len(data) {
			continue
		}
		localized := parseMSGStream(data, start, end, dec)
		var texts []string
		for _, in := range localized {
			if in.Op >= msgLine1 && in.Op <= msgText {
				// FUN_00417010 / sprintf consumes a NUL-terminated string.
				// Some patch entries retain old encrypted padding after it.
				text, _, _ := strings.Cut(in.Text, "\x00")
				texts = append(texts, text)
			}
		}
		j := 0
		for k := range code {
			if code[k].Op >= msgLine1 && code[k].Op <= msgText {
				if j < len(texts) {
					code[k].DisplayText = texts[j]
				}
				j++
			}
		}
	}
}

// Open starts entry id (ECL 338's FUN_00415db0); nil when there is no such
// entry.
func (s *StageMSG) Open(id int32, hooks Hooks) *MSG {
	if s == nil || id < 0 || int(id) >= len(s.Entries) {
		return nil
	}
	m := NewMSG(id, s.Entries[id], hooks)
	m.View.PlayerName = s.PlayerName
	if int(id) < len(s.BossNames) {
		m.View.BossName = s.BossNames[id]
	}
	return m
}

// stageIndex extracts the stage number from a stage key (0 = unknown).
func stageIndex(stage string) int {
	switch stage {
	case "stage1":
		return 1
	case "stage2":
		return 2
	case "stage3":
		return 3
	case "stage4":
		return 4
	case "stage5":
		return 5
	case "stage6":
		return 6
	case "stage7", "extra":
		return 7
	}
	return 0
}
