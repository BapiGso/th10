package menus

import (
	"fmt"
	"path"
	"strings"
	"sync"

	"th10/audio"
	"th10/gamedata"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/encoding/simplifiedchinese"
)

type MusicTrack struct {
	File, Title string
	Comments    [8]string
	BGMIndex    int
}

// parseMusic is FUN_00433ef0's file parser, using FUN_004349e0/00434a20.
// Each @ block has a path, a list title and exactly eight nonempty lines;
// an ideographic space is a real blank comment, not a skipped delimiter.
func parseMusic(data []byte, enc encoding.Encoding) ([]MusicTrack, error) {
	decoded, err := enc.NewDecoder().Bytes(data)
	if err != nil {
		return nil, err
	}
	var lines []string
	for _, line := range strings.Split(string(decoded), "\n") {
		line = strings.TrimSuffix(line, "\r")
		if line != "" {
			lines = append(lines, line)
		}
	}
	var tracks []MusicTrack
	for i := 0; i < len(lines); {
		if !strings.HasPrefix(lines[i], "@") {
			i++ // comments and unrelated lines, 0x433fe0..0x43404c
			continue
		}
		if i+9 >= len(lines) || len(tracks) >= 32 {
			return nil, fmt.Errorf("music room: truncated/oversized comment block")
		}
		track := MusicTrack{File: strings.TrimPrefix(lines[i], "@"), Title: lines[i+1], BGMIndex: -1}
		for n := range track.Comments {
			if strings.HasPrefix(lines[i+2+n], "@") || strings.HasPrefix(lines[i+2+n], "#") {
				return nil, fmt.Errorf("music room: truncated comment block")
			}
			track.Comments[n] = lines[i+2+n]
		}
		tracks = append(tracks, track)
		i += 10
	}
	if len(tracks) == 0 {
		return nil, fmt.Errorf("music room: no tracks")
	}
	return tracks, nil
}

var musicFiles = sync.OnceValues(func() ([]MusicTrack, error) {
	data, err := gamedata.ReadFile("misc/musiccmt.txt")
	if err != nil {
		return nil, err
	}
	tracks, err := parseMusic(data, japanese.ShiftJIS)
	if err != nil {
		return nil, err
	}
	if data, err := gamedata.ReadFile("misc/musiccmtt.txt"); err == nil {
		if translated, err := parseMusic(data, simplifiedchinese.GBK); err == nil && len(translated) == len(tracks) {
			for i := range tracks {
				if translated[i].File == tracks[i].File {
					tracks[i].Title, tracks[i].Comments = translated[i].Title, translated[i].Comments
				}
			}
		}
	}
	bgm, err := audio.BGMTracks()
	if err != nil {
		return nil, err
	}
	for i := range tracks {
		for n, track := range bgm {
			if path.Base(tracks[i].File) == track.Name {
				tracks[i].BGMIndex = n
				break
			}
		}
		if tracks[i].BGMIndex < 0 {
			return nil, fmt.Errorf("music room: unknown BGM %s", tracks[i].File)
		}
	}
	return tracks, nil
})

// PE 0x476fd8: eight warning lines, not part of musiccmt.txt. ASCII escapes
// preserve the original Shift-JIS text without inventing translated copy.
var musicWarning = [8]string{
	"\u3000",
	"\u3000\u3000\uff0a\uff0a\u9078\u629e\u3057\u305f\u66f2\u306f\u307e\u3060\u30b2\u30fc\u30e0\u4e2d\u3067\u518d\u751f\u3055\u308c\u3066\u3044\u307e\u305b\u3093\uff0a\uff0a",
	" ",
	"\u3000\u3000\u3000\u3000\u66f2\u306e\u30b3\u30e1\u30f3\u30c8\u304c\u30cd\u30bf\u30d0\u30ec\u306b\u306a\u308b\u6050\u308c\u304c\u3042\u308a\u307e\u3059\u3002",
	"\u3000\u3000\u3000\u3000\u3000\u3000\u3000\u3000\u3000\u305d\u308c\u3067\u3082\u518d\u751f\u3057\u307e\u3059\u304b\uff1f",
	"\u3000",
	"\u3000\u3000\u3000\u518d\u751f\u3057\u305f\u3044\u65b9\u306f\u3082\u3046\u4e00\u5ea6\u6c7a\u5b9a\u30dc\u30bf\u30f3\u3092\u62bc\u3057\u3066\u304f\u3060\u3055\u3044",
	"\u3000\u3000\u3000\u518d\u751f\u3057\u305f\u304f\u306a\u3044\u65b9\u306f\u3001\u30ab\u30fc\u30bd\u30eb\u3092\u79fb\u52d5\u3057\u3066\u304f\u3060\u3055\u3044",
}
