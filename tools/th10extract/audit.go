package main

import (
	"bytes"
	"compress/flate"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"th10/anm"
	"th10/datfs"

	gtfont "github.com/go-text/typesetting/font"
)

type assetSize struct {
	Files int   `json:"files"`
	Bytes int64 `json:"bytes"`
}

type textureAudit struct {
	File        string   `json:"anm"`
	Entry       int      `json:"entry"`
	Name        string   `json:"name"`
	Format      int      `json:"format"`
	Width       int      `json:"width"`
	Height      int      `json:"height"`
	Bytes       int      `json:"bytes"`
	Colors      int      `json:"colors"`
	Transparent float64  `json:"transparent"`
	SHA256      string   `json:"sha256"`
	Identical   []string `json:"identical"`
}

type bgmAudit struct {
	Name     string  `json:"name"`
	Bytes    int64   `json:"bytes"`
	Seconds  float64 `json:"seconds"`
	Intro    int64   `json:"intro_frames"`
	End      int64   `json:"end_frames"`
	Probe    string  `json:"ffprobe"`
	Reserved uint32  `json:"reserved_bytes"`
}

type seAudit struct {
	Name     string `json:"name"`
	Bytes    int    `json:"bytes"`
	PCMBytes int    `json:"pcm_bytes"`
	Format   int    `json:"format"`
	Rate     int    `json:"rate"`
	Channels int    `json:"channels"`
	Bits     int    `json:"bits"`
}

type fontAudit struct {
	Name    string         `json:"name"`
	Bytes   int            `json:"bytes"`
	Glyphs  int            `json:"glyphs"`
	Used    int            `json:"used_glyphs"`
	Missing string         `json:"missing"`
	Tables  map[string]int `json:"tables"`
}

type assetAudit struct {
	Categories map[string]assetSize `json:"categories"`
	Total      int64                `json:"total"`
	Textures   []textureAudit       `json:"textures"`
	BGM        []bgmAudit           `json:"bgm"`
	SE         []seAudit            `json:"se"`
	Fonts      []fontAudit          `json:"fonts"`
	ECLRefs    map[string][]string  `json:"ecl_refs"`
	STDRefs    map[string]string    `json:"std_refs"`
	MSGRefs    map[string][]string  `json:"msg_refs"`
	Characters fontCharacters       `json:"characters"`
}

func auditAssets(root, dst string) error {
	a := assetAudit{
		Categories: map[string]assetSize{}, ECLRefs: map[string][]string{},
		STDRefs: map[string]string{}, MSGRefs: map[string][]string{},
	}
	duplicates := map[string][]string{}
	var losslessPack []byte
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		st, err := d.Info()
		if err != nil {
			return err
		}
		category := strings.SplitN(rel, "/", 2)[0]
		if filepath.Ext(p) == ".go" {
			return nil
		}
		s := a.Categories[category]
		s.Files++
		s.Bytes += st.Size()
		a.Categories[category], a.Total = s, a.Total+st.Size()
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		switch filepath.Ext(p) {
		case ".anm", ".ant":
			if anm.IsPacked(data) {
				if losslessPack == nil {
					losslessPack, err = os.ReadFile(filepath.Join(root, "textures", "lossless.tpk"))
					if err != nil {
						return err
					}
				}
				data, err = anm.Restore(data, func(id string) ([]byte, error) { return anm.TextureBlob(losslessPack, id) })
				if err != nil {
					return err
				}
			}
			f, err := anm.Load(data)
			if err != nil {
				return fmt.Errorf("%s: %w", p, err)
			}
			for i, e := range f.Entries {
				if e.Texture == nil {
					continue
				}
				t := e.Texture
				h := sha256.New()
				_, _ = fmt.Fprintf(h, "%d/%d/%d/", t.Format, t.W, t.H)
				_, _ = h.Write(t.Pixels)
				hash := hex.EncodeToString(h.Sum(nil))
				ref := fmt.Sprintf("%s#%d", filepath.Base(p), i)
				duplicates[hash] = append(duplicates[hash], ref)
				colors := map[uint32]struct{}{}
				transparent := 0
				pix := t.Image().Pix
				for at := 0; at < len(pix); at += 4 {
					colors[binary.LittleEndian.Uint32(pix[at:])] = struct{}{}
					if pix[at+3] == 0 {
						transparent++
					}
				}
				a.Textures = append(a.Textures, textureAudit{
					File: filepath.Base(p), Entry: i, Name: e.Name, Format: t.Format,
					Width: t.W, Height: t.H, Bytes: len(t.Pixels), Colors: len(colors),
					Transparent: float64(transparent) / float64(t.W*t.H), SHA256: hash,
				})
			}
		case ".wav":
			if len(data) >= 8 && string(data[:4]) == "ZSE1" {
				size := uint64(binary.LittleEndian.Uint32(data[4:]))
				if size > 16<<20 {
					return fmt.Errorf("%s: oversized SE", p)
				}
				r := flate.NewReader(bytes.NewReader(data[8:]))
				data, err = io.ReadAll(io.LimitReader(r, int64(size)+1))
				_ = r.Close()
				if err != nil || uint64(len(data)) != size {
					return fmt.Errorf("%s: invalid SE envelope", p)
				}
			}
			s, err := auditWAV(data)
			if err != nil {
				return fmt.Errorf("%s: %w", p, err)
			}
			s.Name, s.Bytes = filepath.Base(p), int(st.Size())
			a.SE = append(a.SE, s)
		case ".ttc", ".ttf":
			f, err := auditFont(data)
			if err != nil {
				return fmt.Errorf("%s: %w", p, err)
			}
			f.Name, f.Bytes = filepath.Base(p), len(data)
			a.Fonts = append(a.Fonts, f)
		case ".ecl":
			names, err := auditECLNames(data)
			if err != nil {
				return fmt.Errorf("%s: %w", p, err)
			}
			a.ECLRefs[filepath.Base(p)] = names
		case ".std":
			if len(data) >= 0x90 {
				a.STDRefs[filepath.Base(p)] = cString(data[0x10:0x90])
			}
		case ".msg":
			if strings.HasPrefix(filepath.Base(p), "e") || filepath.Base(p) == "staff.msg" {
				a.MSGRefs[filepath.Base(p)] = auditMSGNames(data)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	for i := range a.Textures {
		a.Textures[i].Identical = duplicates[a.Textures[i].SHA256]
	}
	fmtData, err := os.ReadFile(filepath.Join(root, "wav", "bgm", "thbgm.fmt"))
	if err != nil {
		return err
	}
	tracks, err := datfs.ParseBGMFmt(fmtData)
	if err != nil {
		return err
	}
	for _, t := range tracks {
		p := filepath.Join(root, filepath.FromSlash(datfs.BGMPath(t.Name)))
		st, err := os.Stat(p)
		if err != nil {
			return err
		}
		probe, err := exec.Command("ffprobe", "-v", "error", "-show_entries",
			"stream=codec_name,sample_rate,channels,bit_rate,duration:format=duration,bit_rate", "-of", "json", p).Output()
		if err != nil {
			return fmt.Errorf("ffprobe %s: %w", p, err)
		}
		frameSize := int64(t.Channels * t.BitsPerSample / 8)
		a.BGM = append(a.BGM, bgmAudit{
			Name: t.Name, Bytes: st.Size(), Intro: t.Intro / frameSize, End: t.Size / frameSize,
			Seconds: float64(t.Size) / float64(frameSize*int64(t.SampleRate)), Probe: string(probe), Reserved: t.Unknown,
		})
	}
	if a.Characters, err = collectFontCharacters(root); err != nil {
		return err
	}
	for i := range a.Fonts {
		f := &a.Fonts[i]
		data, err := os.ReadFile(filepath.Join(root, "font", f.Name))
		if err != nil {
			return err
		}
		collection, err := gtfont.ParseTTC(bytes.NewReader(data))
		if err != nil {
			return fmt.Errorf("%s: %w", f.Name, err)
		}
		font := collection[0]
		text := a.Characters.Text
		if f.Name == "Bimini Bold.ttf" {
			text = a.Characters.Sources["ASCII"]
		}
		glyphs := map[gtfont.GID]bool{}
		for _, r := range text {
			g, ok := font.NominalGlyph(r)
			if !ok || g == 0 {
				f.Missing += string(r)
			} else {
				glyphs[g] = true
			}
		}
		f.Used = len(glyphs)
	}
	data, err := json.MarshalIndent(a, "", "  ")
	if err != nil {
		return err
	}
	return write(dst, append(data, '\n'))
}

func auditWAV(data []byte) (seAudit, error) {
	var s seAudit
	if len(data) < 12 || string(data[:4]) != "RIFF" || string(data[8:12]) != "WAVE" {
		return s, fmt.Errorf("not a WAV")
	}
	for at := 12; at+8 <= len(data); {
		n := int(binary.LittleEndian.Uint32(data[at+4:]))
		start, end := at+8, at+8+n
		if n < 0 || end > len(data) {
			return s, fmt.Errorf("truncated WAV chunk")
		}
		switch string(data[at : at+4]) {
		case "fmt ":
			if n < 16 {
				return s, fmt.Errorf("short WAV format")
			}
			s.Format = int(binary.LittleEndian.Uint16(data[start:]))
			s.Channels = int(binary.LittleEndian.Uint16(data[start+2:]))
			s.Rate = int(binary.LittleEndian.Uint32(data[start+4:]))
			s.Bits = int(binary.LittleEndian.Uint16(data[start+14:]))
		case "data":
			s.PCMBytes += n
		}
		at = end + n%2
	}
	return s, nil
}

func auditFont(data []byte) (fontAudit, error) {
	f := fontAudit{Tables: map[string]int{}}
	base := 0
	if len(data) >= 16 && string(data[:4]) == "ttcf" {
		base = int(binary.BigEndian.Uint32(data[12:]))
	}
	if base < 0 || base+12 > len(data) {
		return f, fmt.Errorf("truncated SFNT")
	}
	n := int(binary.BigEndian.Uint16(data[base+4:]))
	for i := range n {
		at := base + 12 + i*16
		if at+16 > len(data) {
			return f, fmt.Errorf("truncated SFNT table")
		}
		tag := string(data[at : at+4])
		off, size := int(binary.BigEndian.Uint32(data[at+8:])), int(binary.BigEndian.Uint32(data[at+12:]))
		if off < 0 || size < 0 || off+size > len(data) {
			return f, fmt.Errorf("invalid SFNT table")
		}
		f.Tables[tag] = size
		if tag == "maxp" && size >= 6 {
			f.Glyphs = int(binary.BigEndian.Uint16(data[off+4:]))
		}
	}
	return f, nil
}

func cString(data []byte) string {
	s, _, _ := bytes.Cut(data, []byte{0})
	return string(s)
}

func auditECLNames(data []byte) ([]string, error) {
	if len(data) < 36 || string(data[:4]) != "SCPT" {
		return nil, fmt.Errorf("not an ECL")
	}
	at := int(binary.LittleEndian.Uint32(data[8:]))
	var out []string
	for _, magic := range []string{"ANIM", "ECLI"} {
		if at+8 > len(data) || string(data[at:at+4]) != magic {
			return nil, fmt.Errorf("missing %s", magic)
		}
		n := int(binary.LittleEndian.Uint32(data[at+4:]))
		at += 8
		for range n {
			if at >= len(data) {
				return nil, fmt.Errorf("truncated %s", magic)
			}
			s := cString(data[at:])
			out = append(out, s)
			at += len(s) + 1
		}
		at = (at + 3) &^ 3
	}
	return out, nil
}

func auditMSGNames(data []byte) []string {
	if len(data) < 12 {
		return nil
	}
	var out []string
	at := int(binary.LittleEndian.Uint32(data[4:]))
	for at+4 <= len(data) {
		op, n := data[at+2], int(data[at+3])
		if at+4+n > len(data) {
			break
		}
		args := data[at+4 : at+4+n]
		if op == 7 || op == 10 || op == 12 {
			if op == 7 && n > 4 {
				args = args[4:]
			}
			s := cString(args)
			if s != "" {
				out = append(out, s)
			}
		}
		at += n + 4
	}
	sort.Strings(out)
	return out
}
