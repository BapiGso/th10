// Package assets embeds the game's resources: unmodified copies of th10.dat
// files (extracted with tools/th10extract; layout in datfs.EmbedPath),
// decoded copies of its textures under anm/, thbgm.dat's BGM as Ogg Vorbis,
// and the self-made HUD font.
package assets

import "embed"

// DFYuGaSo TTC 因为简中缺字（"灵丽红选关开" 等都是 .notdef）已经不再嵌入；
// 仍保留磁盘文件以便日后查阅，仓库 .gitignore 没碰它们。
// HUD 数字字体走 Bimini Bold（嵌入）；菜单/对话用 OS 黑体（运行时加载）。
//
//go:embed anm ecl msg sht "font/Bimini Bold.ttf" wav/se/*.wav wav/bgm/*.ogg wav/bgm/thbgm.fmt
var Assets embed.FS
