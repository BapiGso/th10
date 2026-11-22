package assets

import (
	"encoding/binary"
	"errors"
	"fmt"
	"path"
	"strings"
)

// Web release asset packs. The js release build (tag th10web) embeds no
// resources; tools/wasmbuild writes the runtime files into these packs and
// the game fetches them on demand. Desktop and the default js build keep
// embedding everything, so this layout never affects logic or traces.

// PackMagic starts every pack: "TAP1", entry count, then entries of
// kind (0 file, 1 texture blob), u16 name length, name, u32 length, data.
const PackMagic = "TAP1"

const (
	PackFile    = 0
	PackTexture = 1
)

// PackEntry is one file or one texture blob inside a pack.
type PackEntry struct {
	Kind byte
	Name string // assets-relative path, or texture ID
	Data []byte
}

// EncodePack serializes entries in the given order.
func EncodePack(entries []PackEntry) ([]byte, error) {
	size := 8
	for _, e := range entries {
		if len(e.Name) > 0xffff || uint64(len(e.Data)) > 0xffffffff {
			return nil, fmt.Errorf("assets: pack entry %q too large", e.Name)
		}
		size += 7 + len(e.Name) + len(e.Data)
	}
	out := make([]byte, 8, size)
	copy(out, PackMagic)
	binary.LittleEndian.PutUint32(out[4:], uint32(len(entries)))
	for _, e := range entries {
		out = append(out, e.Kind)
		out = binary.LittleEndian.AppendUint16(out, uint16(len(e.Name)))
		out = append(out, e.Name...)
		out = binary.LittleEndian.AppendUint32(out, uint32(len(e.Data)))
		out = append(out, e.Data...)
	}
	return out, nil
}

// DecodePack returns entries whose Data alias data.
func DecodePack(data []byte) ([]PackEntry, error) {
	if len(data) < 8 || string(data[:4]) != PackMagic {
		return nil, errors.New("assets: not an asset pack")
	}
	n := int(binary.LittleEndian.Uint32(data[4:]))
	entries := make([]PackEntry, 0, min(n, len(data)/7))
	at := 8
	for range n {
		if len(data)-at < 3 {
			return nil, errors.New("assets: truncated pack entry")
		}
		kind, nameLen := data[at], int(binary.LittleEndian.Uint16(data[at+1:]))
		at += 3
		if len(data)-at < nameLen+4 {
			return nil, errors.New("assets: truncated pack entry name")
		}
		name := string(data[at : at+nameLen])
		at += nameLen
		dataLen := int(binary.LittleEndian.Uint32(data[at:]))
		at += 4
		if dataLen < 0 || len(data)-at < dataLen {
			return nil, errors.New("assets: truncated pack entry data")
		}
		if kind != PackFile && kind != PackTexture {
			return nil, fmt.Errorf("assets: unknown pack entry kind %d", kind)
		}
		entries = append(entries, PackEntry{Kind: kind, Name: name, Data: data[at : at+dataLen : at+dataLen]})
		at += dataLen
	}
	if at != len(data) {
		return nil, errors.New("assets: trailing bytes after pack entries")
	}
	return entries, nil
}

// ManifestPath is fetched relative to the game document, uncached; packs
// carry their content hash in the file name and may be cached forever.
const ManifestPath = "assets/manifest.json"

// ManifestPack describes one published pack.
type ManifestPack struct {
	File   string `json:"file"`
	Bytes  int    `json:"bytes"`
	SHA256 string `json:"sha256"`
}

// Manifest is written by tools/wasmbuild next to the packs.
type Manifest struct {
	Packs map[string]ManifestPack `json:"packs"`
}

// BootPack holds everything read before the title screen can run: the
// loading and title ANMs, SE (preloaded by the audio manager), the BGM loop
// table and the title theme (warmed before the first frame).
const BootPack = "boot"

// PackOf assigns a runtime file to its pack. Textures travel with the ANM
// that references them. Unlisted files fall back to "common".
func PackOf(name string) string {
	base := path.Base(name)
	stem := strings.TrimSuffix(base, path.Ext(base))
	switch {
	case strings.HasPrefix(name, "wav/bgm/") && path.Ext(name) == ".ogg":
		if stem == bgmFile[0] {
			return BootPack
		}
		return "bgm-" + stem
	case strings.HasPrefix(name, "wav/"), strings.HasPrefix(name, "misc/"):
		return BootPack
	case strings.HasPrefix(name, "anm/"):
		switch stem {
		case "sig", "ascii", "title", "title_v", "text", "bullet":
			return BootPack
		case "staff":
			return "ending"
		}
		if len(stem) == 3 && stem[0] == 'e' {
			return "ending"
		}
		for _, prefix := range []string{"stage", "stgenm", "st"} {
			if n, ok := strings.CutPrefix(stem, prefix); ok && len(n) >= 2 && n[0] == '0' && n[1] >= '1' && n[1] <= '7' {
				return "stage0" + n[1:2]
			}
		}
	case strings.HasPrefix(name, "ecl/stage"), strings.HasPrefix(name, "std/stage"):
		if n := strings.TrimPrefix(stem, "stage"); len(n) == 2 && n[0] == '0' && n[1] >= '1' && n[1] <= '7' {
			return "stage" + n
		}
	case strings.HasPrefix(name, "msg/"):
		if stem == "staff" || (len(stem) == 3 && stem[0] == 'e') {
			return "ending"
		}
		if len(stem) >= 4 && stem[:3] == "st0" && stem[3] >= '1' && stem[3] <= '7' {
			return "stage0" + stem[3:4]
		}
	}
	return "common"
}

// bgmFile maps thbgm.fmt record indexes (audio.BGM*) to their embedded
// file stems; the fmt order is not the file-name order.
var bgmFile = [18]string{
	"th10_02", "th10_00", "th10_01", "th10_03", "th10_04", "th10_05", "th10_06", "th10_07", "th10_08",
	"th10_09", "th10_10", "th10_11", "th10_12", "th10_15", "th10_16", "th10_13", "th10_14", "th10_17",
}

// PackOrder is the background prefetch order after the boot pack: shared
// gameplay files, then each stage followed by its road and boss themes, in
// play order. A read of an unloaded pack fetches it ahead of this queue.
func PackOrder() []string {
	order := []string{BootPack, "common"}
	for stage := 1; stage <= 6; stage++ {
		order = append(order, fmt.Sprintf("stage%02d", stage), "bgm-"+bgmFile[stage*2-1], "bgm-"+bgmFile[stage*2])
	}
	order = append(order, "ending", "bgm-"+bgmFile[15], "bgm-"+bgmFile[16], "bgm-"+bgmFile[17])
	return append(order, "stage07", "bgm-"+bgmFile[13], "bgm-"+bgmFile[14])
}
