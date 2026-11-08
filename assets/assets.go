// Package assets embeds the game's resources: unmodified copies of th10.dat
// files (extracted with tools/th10extract; layout in datfs.EmbedPath),
// decoded copies of its textures under anm/, thbgm.dat's BGM as Ogg Vorbis,
// and the self-made HUD font.
package assets

import "embed"

// GUI/MSG text uses the existing DFYuGaSo fonts with th10c.dat's traditional
// Chinese text. Menus keep their current font policy; numeric GUI sprites
// come from front.anm/ascii.anm rather than a font.
//
//go:embed anm ecl msg sht std "font/Bimini Bold.ttf" font/DFYuGaSo-W3.ttc font/DFYuGaSo-W7.ttc misc/demo*.rpy misc/musiccmt.txt misc/musiccmtt.txt wav/se/*.wav wav/bgm/*.ogg wav/bgm/thbgm.fmt
var Assets embed.FS
