// Command th10extract generates embedded runtime resources from a TH10
// install. Ordinary files keep their original bytes. -pack-textures splits
// ANM logic from deduplicated lossless THTX pixels; -web-textures additionally
// selects bounded-error web colors. -pack-se losslessly packs WAVs. -fonts
// preserves the original glyph programs for fixed text and printable ASCII.
// -bgm cuts exactly thbgm.fmt's PCM span and uses ffmpeg libvorbis, without
// shifting its sample-exact loop points. -textures is the legacy PNG repair
// command; decoded reference PNGs are no longer runtime assets.
//
// Usage:
//
//	th10extract [-dir DIR] [-archive th10.dat|th10c.dat] [-out assets] [-list] [PATTERN ...]
//	th10extract [-dir DIR] [-out assets] -pack-textures|-web-textures [-compare DIR]
//	th10extract [-dir DIR] [-out assets] -bgm [-q 1] [-bgm-track N]
//	th10extract [-dir DIR] [-out assets] -pack-se
//	th10extract [-out assets] -fonts -font-source DIR
//	th10extract [-out assets] -audit FILE.json
//
// PATTERNs are path.Match globs over th10.dat names ("stage01.std",
// "se_*.wav"). -list prints every th10.dat file and whether it is embedded.
// DIR defaults to $TH10_DIR, then a .tmp/[th10] above the working directory.
package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"th10/anm"
	"th10/datfs"
)

func main() {
	dir := flag.String("dir", "", "TH10 install holding th10.dat and thbgm.dat")
	archive := flag.String("archive", "th10.dat", "resource archive within DIR (th10.dat or th10c.dat)")
	out := flag.String("out", "assets", "assets directory to write into")
	list := flag.Bool("list", false, "list th10.dat and mark what is embedded")
	bgm := flag.Bool("bgm", false, "encode every thbgm.dat track (needs ffmpeg with libvorbis)")
	quality := flag.Float64("q", 1, "Ogg Vorbis quality for -bgm (1 is nominally 80 kbps stereo)")
	bgmTrack := flag.Int("bgm-track", -1, "encode only this zero-based thbgm.fmt track for auditioning")
	se := flag.Bool("pack-se", false, "losslessly compress embedded SE WAVs from th10.dat")
	fonts := flag.Bool("fonts", false, "subset original fonts to fixed game text and printable ASCII")
	fontSource := flag.String("font-source", "", "directory containing unmodified source fonts for -fonts")
	textures := flag.Bool("textures", false, "redraw the decoded pictures under anm/ from th10.dat")
	audit := flag.String("audit", "", "write an asset census JSON (reads -out, does not change assets)")
	pack := flag.Bool("pack-textures", false, "split runtime ANM pixels into deduplicated lossless texture packs")
	web := flag.Bool("web-textures", false, "also choose lossy js textures (implies -pack-textures)")
	compare := flag.String("compare", "", "directory for the worst web texture comparison images")
	flag.Parse()
	if *audit != "" {
		if err := auditAssets(*out, *audit); err != nil {
			fmt.Fprintln(os.Stderr, "th10extract:", err)
			os.Exit(1)
		}
		return
	}
	if *fonts {
		if err := subsetFonts(*out, *fontSource); err != nil {
			fmt.Fprintln(os.Stderr, "th10extract:", err)
			os.Exit(1)
		}
		return
	}
	if *pack || *web {
		if err := packTextures(*dir, *out, *web, *compare); err != nil {
			fmt.Fprintln(os.Stderr, "th10extract:", err)
			os.Exit(1)
		}
		return
	}
	if *se {
		if err := packSounds(*dir, *out); err != nil {
			fmt.Fprintln(os.Stderr, "th10extract:", err)
			os.Exit(1)
		}
		return
	}
	if *bgm && *bgmTrack >= 0 {
		if err := encodeOneBGM(*dir, *archive, *out, *quality, *bgmTrack); err != nil {
			fmt.Fprintln(os.Stderr, "th10extract:", err)
			os.Exit(1)
		}
		return
	}
	if !*list && !*bgm && !*textures && flag.NArg() == 0 {
		flag.Usage()
		os.Exit(2)
	}
	if err := runArchive(*dir, *archive, *out, *list, *bgm, *textures, *quality, flag.Args()); err != nil {
		fmt.Fprintln(os.Stderr, "th10extract:", err)
		os.Exit(1)
	}
}

func run(dir, out string, list, bgm, textures bool, quality float64, patterns []string) error {
	return runArchive(dir, "th10.dat", out, list, bgm, textures, quality, patterns)
}

func runArchive(dir, archive, out string, list, bgm, textures bool, quality float64, patterns []string) error {
	if archive != "th10.dat" && archive != "th10c.dat" {
		return fmt.Errorf("unsupported resource archive %q", archive)
	}
	if dir == "" {
		var err error
		if dir, err = datfs.Locate(); err != nil {
			return err
		}
	}
	dat, err := datfs.Open(filepath.Join(dir, archive))
	if err != nil {
		return err
	}
	defer func() { _ = dat.Close() }()

	if list {
		for _, name := range dat.Names() {
			st, err := dat.Stat(name)
			if err != nil {
				return err
			}
			mark := ""
			if _, err := os.Stat(filepath.Join(out, filepath.FromSlash(datfs.EmbedPath(name)))); err == nil {
				mark = "embedded"
			}
			fmt.Printf("%-20s %9d  %s\n", name, st.Size(), mark)
		}
	}

	for _, pat := range patterns {
		if _, err := path.Match(pat, ""); err != nil {
			return fmt.Errorf("bad pattern %q: %w", pat, err)
		}
		n := 0
		for _, name := range dat.Names() {
			if ok, _ := path.Match(pat, name); !ok {
				continue
			}
			if err := extract(dat, name, out); err != nil {
				return err
			}
			n++
		}
		if n == 0 {
			return fmt.Errorf("%q matches nothing in %s", pat, archive)
		}
	}

	if textures {
		if err := redrawTextures(dat, out); err != nil {
			return err
		}
	}
	if bgm {
		return encodeBGM(dat, dir, out, quality)
	}
	return nil
}

func extract(dat *datfs.Archive, name, out string) error {
	data, err := dat.ReadFile(name)
	if err != nil {
		return err
	}
	return write(filepath.Join(out, filepath.FromSlash(datfs.EmbedPath(name))), data)
}

func write(dst string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		return err
	}
	fmt.Printf("wrote %s (%d bytes)\n", dst, len(data))
	return nil
}

func encodeBGM(dat *datfs.Archive, dir, out string, quality float64) error {
	return encodeBGMSelection(dat, dir, out, quality, -1)
}

func encodeBGMSelection(dat *datfs.Archive, dir, out string, quality float64, only int) error {
	fmtData, err := dat.ReadFile("thbgm.fmt")
	if err != nil {
		return err
	}
	tracks, err := datfs.ParseBGMFmt(fmtData)
	if err != nil {
		return err
	}
	if only < -1 || only >= len(tracks) {
		return fmt.Errorf("BGM track %d out of range", only)
	}
	f, err := os.Open(filepath.Join(dir, "thbgm.dat"))
	if errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("-bgm needs thbgm.dat in %s", dir)
	} else if err != nil {
		return err
	}
	defer f.Close()

	for i, t := range tracks {
		if only >= 0 && only != i {
			continue
		}
		if t.BitsPerSample != 16 {
			return fmt.Errorf("%s: %d-bit PCM, want 16", t.Name, t.BitsPerSample)
		}
		dst := filepath.Join(out, filepath.FromSlash(datfs.BGMPath(t.Name)))
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		cmd := exec.Command("ffmpeg", "-hide_banner", "-loglevel", "error", "-y",
			"-f", "s16le", "-ar", strconv.Itoa(t.SampleRate), "-ac", strconv.Itoa(t.Channels), "-i", "-",
			"-c:a", "libvorbis", "-q:a", strconv.FormatFloat(quality, 'g', -1, 64),
			"-map_metadata", "-1", "-flags", "+bitexact", "-fflags", "+bitexact", dst)
		cmd.Stdin = t.PCM(f)
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("ffmpeg %s: %w", t.Name, err)
		}
		st, err := os.Stat(dst)
		if err != nil {
			return err
		}
		fmt.Printf("wrote %s (%d bytes, %s)\n", dst, st.Size(), t.Name)
	}
	return extract(dat, "thbgm.fmt", out)
}

// redrawTextures rewrites each out/anm/**/*.png that names a th10.dat
// texture and whose pixels (ignoring colour under zero alpha) differ from it.
func redrawTextures(dat *datfs.Archive, out string) error {
	textures := map[string]*anm.File{} // first .anm in archive order wins
	for _, name := range dat.Names() {
		if !strings.HasSuffix(name, ".anm") {
			continue
		}
		data, err := dat.ReadFile(name)
		if err != nil {
			return err
		}
		f, err := anm.Load(data)
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		for _, e := range f.Entries {
			if _, seen := textures[e.Name]; !seen && e.Texture != nil {
				textures[e.Name] = f
			}
		}
	}
	root := filepath.Join(out, "anm")
	return filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".png") {
			return err
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		f, ok := textures[filepath.ToSlash(rel)]
		if !ok {
			return nil
		}
		want := f.NamedImage(filepath.ToSlash(rel))
		old, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		if img, err := png.Decode(bytes.NewReader(old)); err == nil && samePixels(img, want) {
			return nil
		}
		var buf bytes.Buffer
		if err := png.Encode(&buf, want); err != nil {
			return err
		}
		return write(p, buf.Bytes())
	})
}

func samePixels(a image.Image, b *image.NRGBA) bool {
	if a.Bounds() != b.Bounds() {
		return false
	}
	n := image.NewNRGBA(b.Rect)
	draw.Draw(n, n.Rect, a, a.Bounds().Min, draw.Src)
	for i := 0; i < len(n.Pix); i += 4 {
		if n.Pix[i+3] == 0 && b.Pix[i+3] == 0 {
			continue
		}
		if !bytes.Equal(n.Pix[i:i+4], b.Pix[i:i+4]) {
			return false
		}
	}
	return true
}
