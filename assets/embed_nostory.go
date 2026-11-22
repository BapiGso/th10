//go:build !th10web && th10nostory

package assets

import "embed"

// Assets without dialogue and endings (th10nostory): no ending or staff-roll
// ANM/MSG (e??, staff) and no .mst texts. Portrait sheets stay (pl0N is also
// the player, stgenm0N the spell cut-ins) and so do the stage .msg scripts,
// which drive stage logic.
//
//go:embed anm/[^e]*.anm anm/enemy.anm anm/[^s]*.ant anm/st0*.ant ecl/*.ecl ecl/*.ect msg/st*.msg sht/*.sht std/*.std font/DFYuGaSo-W3.ttc font/DFYuGaSo-W7.ttc misc/demo*.rpy misc/musiccmt.txt misc/musiccmtt.txt wav/se/*.wav wav/bgm/*.ogg wav/bgm/thbgm.fmt
var Assets embed.FS
