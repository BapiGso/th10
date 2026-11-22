//go:build !th10web

package assets

import (
	"bytes"
	"io/fs"
	"strings"
	"testing"
)

func TestPackRoundTrip(t *testing.T) {
	in := []PackEntry{{PackFile, "anm/a.anm", []byte("abc")}, {PackTexture, "deadbeef", nil}, {PackFile, "x", bytes.Repeat([]byte{7}, 70000)}}
	data, err := EncodePack(in)
	if err != nil {
		t.Fatal(err)
	}
	out, err := DecodePack(data)
	if err != nil || len(out) != len(in) {
		t.Fatalf("decode: %v %d", err, len(out))
	}
	for i := range in {
		if out[i].Kind != in[i].Kind || out[i].Name != in[i].Name || !bytes.Equal(out[i].Data, in[i].Data) {
			t.Fatalf("entry %d differs", i)
		}
	}
	for cut := range len(data) - 1 {
		if _, err := DecodePack(data[:cut]); err == nil {
			t.Fatalf("truncated pack at %d accepted", cut)
		}
	}
	if _, err := DecodePack(append(data, 0)); err == nil {
		t.Fatal("trailing byte accepted")
	}
}

// bgmFile must follow thbgm.fmt's record order, which audio.BGM* indexes.
func TestBGMFileOrderMatchesFmt(t *testing.T) {
	data, err := Assets.ReadFile("wav/bgm/thbgm.fmt")
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for field := range bytes.SplitSeq(data, []byte{0}) {
		if s := string(field); strings.HasPrefix(s, "th10_") && strings.HasSuffix(s, ".wav") {
			names = append(names, strings.TrimSuffix(s, ".wav"))
		}
	}
	if len(names) != len(bgmFile) {
		t.Fatalf("thbgm.fmt has %d tracks", len(names))
	}
	for i, name := range names {
		if bgmFile[i] != name {
			t.Fatalf("record %d is %s, table says %s", i, name, bgmFile[i])
		}
	}
}

func TestPackLayout(t *testing.T) {
	used := map[string]int{}
	err := fs.WalkDir(Assets, ".", func(name string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			used[PackOf(name)]++
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	order := map[string]bool{}
	for _, name := range PackOrder() {
		if order[name] {
			t.Fatalf("pack %s listed twice", name)
		}
		order[name] = true
		if used[name] == 0 {
			t.Fatalf("prefetch order lists empty pack %s", name)
		}
	}
	for name := range used {
		if !order[name] {
			t.Errorf("pack %s is never prefetched", name)
		}
	}
	// Read before the title runs: loading, title, SE preload, Vorbis prewarm.
	for _, name := range []string{"anm/sig.anm", "anm/ascii.anm", "anm/title.anm", "anm/title.ant", "anm/title_v.anm", "anm/text.anm",
		"anm/bullet.anm", "wav/se/se_ok00.wav", "wav/bgm/thbgm.fmt", "wav/bgm/th10_02.ogg", "misc/demo0.rpy"} {
		if PackOf(name) != BootPack {
			t.Errorf("%s is in %s, not boot", name, PackOf(name))
		}
	}
	for name, want := range map[string]string{"anm/st03logo.ant": "stage03", "anm/stgenm07.anm": "stage07", "ecl/stage05.ect": "stage05",
		"std/stage01.std": "stage01", "msg/st06_01.mst": "stage06", "msg/e11.msg": "ending", "anm/staff.ant": "ending", "anm/e00.anm": "ending",
		"ecl/default.ecl": "common", "anm/front.anm": "common", "sht/pl01c.sht": "common", "wav/bgm/th10_17.ogg": "bgm-th10_17"} {
		if got := PackOf(name); got != want {
			t.Errorf("PackOf(%s) = %s, want %s", name, got, want)
		}
	}
}
