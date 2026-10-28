package datfs

import (
	"path"
	"strings"
)

// EmbedPath is where th10.dat's file datName is embedded, relative to
// assets/ (see package gamedata).
func EmbedPath(datName string) string {
	switch ext := path.Ext(datName); ext {
	case ".ecl", ".msg", ".sht", ".anm", ".std":
		return ext[1:] + "/" + datName
	case ".wav":
		return "wav/se/" + datName
	case ".fmt":
		return "wav/bgm/" + datName
	}
	return "misc/" + datName
}

// BGMPath is where the thbgm.dat track recorded in thbgm.fmt as wavName
// ("th10_02.wav") is embedded, encoded as Ogg Vorbis.
func BGMPath(wavName string) string {
	return "wav/bgm/" + strings.TrimSuffix(wavName, ".wav") + ".ogg"
}
