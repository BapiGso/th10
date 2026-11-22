package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/encoding/simplifiedchinese"
)

type fontCharacters struct {
	Text    string            `json:"text"`
	Count   int               `json:"count"`
	Sources map[string]string `json:"sources"`
}

// collectFontCharacters covers both original and translated fixed text, plus
// printable ASCII for user names. Source literals are a conservative closure
// for menus, warning messages and the names supplied to the legacy Result UI.
func collectFontCharacters(root string) (fontCharacters, error) {
	sets := map[string]map[rune]bool{}
	add := func(source, text string) {
		if sets[source] == nil {
			sets[source] = map[rune]bool{}
		}
		for _, r := range text {
			if unicode.IsPrint(r) {
				sets[source][r] = true
			}
		}
	}
	for r := rune(32); r <= 126; r++ {
		add("ASCII", string(r))
	}
	for _, folder := range []string{"msg", "ecl", "misc"} {
		files, err := os.ReadDir(filepath.Join(root, folder))
		if err != nil {
			return fontCharacters{}, err
		}
		for _, entry := range files {
			name := entry.Name()
			ext := filepath.Ext(name)
			if ext != ".msg" && ext != ".mst" && ext != ".ecl" && ext != ".ect" && name != "musiccmt.txt" && name != "musiccmtt.txt" {
				continue
			}
			data, err := os.ReadFile(filepath.Join(root, folder, name))
			if err != nil {
				return fontCharacters{}, err
			}
			enc := encoding.Encoding(japanese.ShiftJIS)
			if ext == ".mst" || ext == ".ect" || name == "musiccmtt.txt" {
				enc = simplifiedchinese.GBK
			}
			category := "MSG"
			if folder == "misc" {
				text, err := enc.NewDecoder().Bytes(data)
				if err != nil {
					return fontCharacters{}, err
				}
				add("Music", string(text))
				continue
			}
			var texts []string
			if folder == "ecl" {
				category = "Spell"
				texts, err = fontECLStrings(data, enc)
			} else {
				opcodes := map[byte]bool{14: true, 15: true, 16: true}
				if strings.HasPrefix(name, "e") || strings.HasPrefix(name, "staff") {
					category = "Ending"
					if strings.HasPrefix(name, "staff") {
						category = "Staff"
					}
					opcodes = map[byte]bool{3: true}
				}
				texts, err = fontMSGStrings(data, opcodes, enc)
			}
			if err != nil {
				return fontCharacters{}, fmt.Errorf("%s: %w", name, err)
			}
			for _, text := range texts {
				add(category, text)
			}
		}
	}
	project := filepath.Dir(filepath.Clean(root))
	for _, folder := range []string{"game", "audio", "scene"} {
		err := filepath.WalkDir(filepath.Join(project, folder), func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
				return err
			}
			file, err := parser.ParseFile(token.NewFileSet(), p, nil, 0)
			if err != nil {
				return err
			}
			ast.Inspect(file, func(n ast.Node) bool {
				if literal, ok := n.(*ast.BasicLit); ok && literal.Kind == token.STRING {
					if text, err := strconv.Unquote(literal.Value); err == nil {
						add("Source", text)
					}
				}
				return true
			})
			return nil
		})
		if err != nil {
			return fontCharacters{}, err
		}
	}
	c := fontCharacters{Sources: map[string]string{}}
	all := map[rune]bool{}
	for name, set := range sets {
		c.Sources[name] = sortedCharacters(set)
		for r := range set {
			all[r] = true
		}
	}
	c.Text, c.Count = sortedCharacters(all), len(all)
	return c, nil
}

func sortedCharacters(set map[rune]bool) string {
	runes := make([]rune, 0, len(set))
	for r := range set {
		runes = append(runes, r)
	}
	sort.Slice(runes, func(i, j int) bool { return runes[i] < runes[j] })
	return string(runes)
}

func decodeFontString(data []byte, enc encoding.Encoding) (string, error) {
	plain := bytes.Clone(data)
	key, step := byte(0x77), byte(7)
	for i := range plain {
		plain[i] ^= key
		key += step
		step += 16
	}
	text, err := enc.NewDecoder().String(cString(plain))
	return text, err
}

func fontMSGStrings(data []byte, opcodes map[byte]bool, enc encoding.Encoding) ([]string, error) {
	if len(data) < 4 {
		return nil, fmt.Errorf("short MSG")
	}
	n := int(binary.LittleEndian.Uint32(data))
	if n < 1 || 4+n*8 > len(data) {
		return nil, fmt.Errorf("invalid MSG count")
	}
	var out []string
	for i := range n {
		start := int(binary.LittleEndian.Uint32(data[4+i*8:]))
		end := len(data)
		if i+1 < n {
			end = int(binary.LittleEndian.Uint32(data[4+(i+1)*8:]))
		}
		if start < 4+n*8 || start > end || end > len(data) {
			return nil, fmt.Errorf("invalid MSG offsets")
		}
		for at := start; at+4 <= end; {
			length := int(data[at+3])
			if at+4+length > end {
				return nil, fmt.Errorf("short MSG arguments")
			}
			if opcodes[data[at+2]] {
				text, err := decodeFontString(data[at+4:at+4+length], enc)
				if err != nil {
					return nil, err
				}
				out = append(out, text)
			}
			at += 4 + length
		}
	}
	return out, nil
}

func fontECLStrings(data []byte, enc encoding.Encoding) ([]string, error) {
	if len(data) < 36 || string(data[:4]) != "SCPT" {
		return nil, fmt.Errorf("invalid ECL")
	}
	n := int(binary.LittleEndian.Uint32(data[16:]))
	at := int(binary.LittleEndian.Uint32(data[8:])) + int(binary.LittleEndian.Uint16(data[6:]))
	if at < 36 || at+n*4 > len(data) {
		return nil, fmt.Errorf("invalid ECL sub table")
	}
	var out []string
	for i := range n {
		start := int(binary.LittleEndian.Uint32(data[at+i*4:])) + 16
		end := len(data)
		if i+1 < n {
			end = int(binary.LittleEndian.Uint32(data[at+(i+1)*4:]))
		}
		if start < 16 || start > end || end > len(data) {
			return nil, fmt.Errorf("invalid ECL sub offsets")
		}
		for pos := start; pos+16 <= end; {
			op, size := int(binary.LittleEndian.Uint16(data[pos+4:])), int(binary.LittleEndian.Uint16(data[pos+6:]))
			if size < 16 || pos+size > end {
				return nil, fmt.Errorf("invalid ECL instruction")
			}
			if op == 342 || op == 348 || op == 357 || op == 358 || op == 359 {
				// SSSx: three integers followed by length and XOR text.
				if size < 32 {
					return nil, fmt.Errorf("short spell declaration")
				}
				length := int(binary.LittleEndian.Uint32(data[pos+28:]))
				if length < 0 || 32+length > size {
					return nil, fmt.Errorf("short spell name")
				}
				text, err := decodeFontString(data[pos+32:pos+32+length], enc)
				if err != nil {
					return nil, err
				}
				out = append(out, text)
			}
			pos += size
		}
	}
	return out, nil
}

func cString(data []byte) string {
	s, _, _ := bytes.Cut(data, []byte{0})
	return string(s)
}
