package ending

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"
	"sync"

	"th10/gamedata"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/encoding/simplifiedchinese"
)

type instruction struct {
	time        uint16
	op          byte
	args        []byte
	text        string
	displayText string
}

func (in instruction) arg(word int) int32 {
	at := word * 4
	if at < 0 || at+4 > len(in.args) {
		return 0
	}
	return int32(binary.LittleEndian.Uint32(in.args[at:]))
}

func cstring(data []byte) string {
	if at := bytes.IndexByte(data, 0); at >= 0 {
		data = data[:at]
	}
	return string(data)
}

// parseMSG reads the independent ending subset of FUN_0040bd80, not the
// stage's opcode meanings. All shipped ending/staff files have entry 0.
func parseMSG(data []byte, enc encoding.Encoding) ([]instruction, error) {
	if len(data) < 12 || binary.LittleEndian.Uint32(data) == 0 {
		return nil, fmt.Errorf("ending: truncated MSG header")
	}
	count := uint64(binary.LittleEndian.Uint32(data))
	start := uint64(binary.LittleEndian.Uint32(data[4:]))
	if count*8+4 > uint64(len(data)) || start < count*8+4 || start > uint64(len(data)-4) {
		return nil, fmt.Errorf("ending: invalid MSG offset")
	}
	var out []instruction
	for at := int(start); at < len(data); {
		if at+4 > len(data) {
			return nil, fmt.Errorf("ending: truncated MSG instruction")
		}
		length := int(data[at+3])
		if at+4+length > len(data) {
			return nil, fmt.Errorf("ending: truncated MSG arguments")
		}
		in := instruction{time: binary.LittleEndian.Uint16(data[at:]), op: data[at+2], args: data[at+4 : at+4+length]}
		if in.op == 3 {
			plain := bytes.Clone(in.args)
			key, step := byte(0x77), byte(7)
			for i := range plain {
				plain[i] ^= key
				key += step
				step += 0x10
			}
			text, err := enc.NewDecoder().Bytes([]byte(cstring(plain)))
			if err != nil {
				return nil, fmt.Errorf("ending: MSG text: %w", err)
			}
			in.text, in.displayText = string(text), string(text) // FUN_00417010
		}
		out = append(out, in)
		at += length + 4
	}
	return out, nil
}

var msgFiles = struct {
	sync.Mutex
	files map[string][]instruction
}{files: map[string][]instruction{}}

func loadMSG(name string) ([]instruction, error) {
	msgFiles.Lock()
	defer msgFiles.Unlock()
	if instructions := msgFiles.files[name]; instructions != nil {
		return instructions, nil
	}
	data, err := gamedata.ReadFile("msg/" + name)
	if err != nil {
		return nil, err
	}
	instructions, err := parseMSG(data, japanese.ShiftJIS)
	if err != nil {
		return nil, err
	}
	// MST is presentation only. Keep original opcodes, times, arguments and
	// Japanese text, and overlay translated lines only when counts agree.
	if data, err := gamedata.ReadFile("msg/" + strings.TrimSuffix(name, ".msg") + ".mst"); err == nil {
		if translated, err := parseMSG(data, simplifiedchinese.GBK); err == nil {
			var lines []string
			count := 0
			for _, in := range translated {
				if in.op == 3 {
					lines = append(lines, in.text)
				}
			}
			for _, in := range instructions {
				if in.op == 3 {
					count++
				}
			}
			if count == len(lines) {
				at := 0
				for i := range instructions {
					if instructions[i].op == 3 {
						instructions[i].displayText = lines[at]
						at++
					}
				}
			}
		}
	}
	msgFiles.files[name] = instructions
	return instructions, nil
}
