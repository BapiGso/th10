package dialog

// TH10 stage dialog (.msg) binary loader.
//
// Faithful to the format thtk decodes: a small instruction VM driving two
// portraits + a textbox. We parse the raw stXX_0C.msg bytes (C = character:
// 00 = Reimu, 01 = Marisa) that th10.exe itself loads, rather than relying on a
// hand-written script table.
//
// On-disk layout (little-endian):
//
//	u32 entry_count
//	entry_count × { u32 offset; u32 id }   (offset = byte offset of the stream)
//	per entry @offset: a stream of instructions until the next entry / EOF
//	instr: i16 time; u8 opcode; u8 length; [length bytes of args]
//
// Opcodes we interpret (names/semantics from thtk th10_msg_fmts + ExpHP's RE,
// cross-checked against the decoded streams):
//
//	7        speakerPlayer — following text lines are the player's (left)
//	8        speakerBoss   — following text lines are the boss's (right)
//	12 S     playerFace    — set the left portrait face id
//	13 S     bossFace      — set the right portrait face id
//	16 m     textAdd       — one dialog line (XOR-encrypted Shift-JIS)
//
// All other opcodes (end/show/hide/wait/music/intro/score…) don't affect the
// text content we surface, so they're skipped.

import (
	"encoding/binary"
	"fmt"
	"sync"

	"th10/assets"

	"golang.org/x/text/encoding/japanese"
)

const (
	msgOpSpeakerPlayer = 7
	msgOpSpeakerBoss   = 8
	msgOpPlayerFace    = 12
	msgOpBossFace      = 13
	msgOpText          = 16
)

// MsgLine is one decoded dialog line: who speaks, their portrait face id, and
// the UTF-8 text.
type MsgLine struct {
	IsRight bool   // true = boss/right speaker, false = player/left
	FaceID  int    // portrait face id from the last 12/13 before this line
	Text    string // decoded UTF-8 text
}

// ParseMSG decodes a TH10 .msg byte slice into ordered dialog lines. Entries
// (conversation segments, e.g. boss-pre / boss-post) are concatenated in file
// order; callers split them via the returned segment boundaries.
func ParseMSG(data []byte) (lines []MsgLine, segments [][]MsgLine) {
	if len(data) < 4 {
		return nil, nil
	}
	count := int(binary.LittleEndian.Uint32(data))
	if count <= 0 || 4+count*8 > len(data) {
		return nil, nil
	}
	offsets := make([]int, count)
	for i := 0; i < count; i++ {
		offsets[i] = int(binary.LittleEndian.Uint32(data[4+i*8:]))
	}

	dec := japanese.ShiftJIS.NewDecoder()
	for i, start := range offsets {
		end := len(data)
		if i+1 < count {
			end = offsets[i+1]
		}
		seg := parseMSGStream(data, start, end, dec)
		if len(seg) > 0 {
			segments = append(segments, seg)
			lines = append(lines, seg...)
		}
	}
	return lines, segments
}

func parseMSGStream(data []byte, p, end int, dec interface {
	Bytes([]byte) ([]byte, error)
}) []MsgLine {
	var out []MsgLine
	isRight := false
	faceL, faceR := 0, 0
	for p+4 <= end {
		op := data[p+2]
		length := int(data[p+3])
		argOff := p + 4
		if argOff+length > end {
			break
		}
		args := data[argOff : argOff+length]
		switch op {
		case msgOpSpeakerPlayer:
			isRight = false
		case msgOpSpeakerBoss:
			isRight = true
		case msgOpPlayerFace:
			if len(args) >= 4 {
				faceL = int(int32(binary.LittleEndian.Uint32(args)))
			}
		case msgOpBossFace:
			if len(args) >= 4 {
				faceR = int(int32(binary.LittleEndian.Uint32(args)))
			}
		case msgOpText:
			if txt := decodeMSGText(args, dec); txt != "" {
				face := faceL
				if isRight {
					face = faceR
				}
				out = append(out, MsgLine{IsRight: isRight, FaceID: face, Text: txt})
			}
		}
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
// Stage-level loader: official .msg → []Line, cached, mapped to our portraits.
// ---------------------------------------------------------------------------

// stageMsgSpeakers names the left (player) and right (boss/enemy) speakers for
// each .msg segment of a stage, so the official text gets our portrait + name.
// Index = segment order in the file. The right speaker varies by segment for
// stages whose mid-boss differs from the boss (e.g. stage 3).
type stageMsgSpeakers struct {
	// rightBySeg[i] = boss/enemy speaker name for segment i (must exist in
	// speakerFaces). Player name is derived from the character at load time.
	rightBySeg []string
}

// msgSpeakers maps stageN → per-segment right-speaker names. Segment layout
// (from the decoded files): most stages = [bossPre, post]; stage 3 has a
// mid-boss = [midPre, midPost, bossPre, post].
var msgSpeakers = map[string]stageMsgSpeakers{
	"stage1": {rightBySeg: []string{"秋静葉", "秋穣子"}}, // mid 静葉(pre), 穣子(boss)
	"stage2": {rightBySeg: []string{"鍵山雛", "鍵山雛"}},
	"stage3": {rightBySeg: []string{"河城にとり", "河城にとり", "河城にとり", "河城にとり"}},
	"stage4": {rightBySeg: []string{"射命丸文", "射命丸文"}},
	"stage5": {rightBySeg: []string{"東風谷早苗", "東風谷早苗"}},
	"stage6": {rightBySeg: []string{"八坂神奈子", "八坂神奈子"}},
	"extra":  {rightBySeg: []string{"八坂神奈子", "洩矢諏訪子", "洩矢諏訪子"}},
}

type stageDialog struct {
	segments [][]Line
}

var (
	msgCache   = map[string]*stageDialog{}
	msgCacheMu sync.Mutex
)

// loadStageDialog parses & caches stXX_0C.msg for a stage ("stage2") and
// character (0=Reimu, 1=Marisa), converting each segment into []Line with our
// speaker names/portraits. Returns nil if the file is missing or unparsable
// (callers then fall back to the hand-written scripts table).
func loadStageDialog(stage string, character int) *stageDialog {
	stageNum := stageIndex(stage)
	if stageNum == 0 {
		return nil
	}
	key := fmt.Sprintf("%s_%d", stage, character&1)
	msgCacheMu.Lock()
	defer msgCacheMu.Unlock()
	if sd, ok := msgCache[key]; ok {
		return sd
	}

	path := fmt.Sprintf("msg/st%02d_%02d.msg", stageNum, character&1)
	data, err := assets.Assets.ReadFile(path)
	if err != nil {
		msgCache[key] = nil
		return nil
	}
	_, rawSegs := ParseMSG(data)
	if len(rawSegs) == 0 {
		msgCache[key] = nil
		return nil
	}

	spk := msgSpeakers[stage]
	playerName := "博麗霊夢"
	if character&1 == 1 {
		playerName = "霧雨魔理沙"
	}

	sd := &stageDialog{}
	for i, seg := range rawSegs {
		right := ""
		if i < len(spk.rightBySeg) {
			right = spk.rightBySeg[i]
		}
		var lines []Line
		for _, ml := range seg {
			speaker := playerName
			if ml.IsRight {
				speaker = right
			}
			lines = append(lines, Line{
				Speaker:    speaker,
				Expression: msgFaceExpression(ml.FaceID),
				Text:       ml.Text,
				IsRight:    ml.IsRight,
			})
		}
		sd.segments = append(sd.segments, lines)
	}
	msgCache[key] = sd
	return sd
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

// msgFaceExpression maps a .msg portrait face id to one of our expression
// suffixes. TH10 uses a compact 0-8 portrait variant index in the instruction
// stream; the concrete image set differs per character, so getFace applies a
// per-suffix fallback chain when a character lacks the exact variant.
func msgFaceExpression(id int) string {
	switch id {
	case 1, 3, 4:
		return "an"
	case 2:
		return "hp"
	case 5:
		return "pr"
	case 6:
		return "n2"
	case 7:
		return "sp"
	case 8:
		return "dp"
	default:
		return "no"
	}
}
