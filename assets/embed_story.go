//go:build !th10web && !th10nostory

package assets

import "embed"

// GUI/MSG text uses the existing DFYuGaSo fonts with th10c.dat's traditional
// Chinese text. Menus keep their current font policy; numeric GUI sprites
// come from front.anm/ascii.anm rather than a font.
//
//go:embed anm/*.anm anm/*.ant ecl/*.ecl ecl/*.ect msg/*.msg msg/*.mst sht/*.sht std/*.std font/DFYuGaSo-W3.ttc font/DFYuGaSo-W7.ttc misc/demo*.rpy misc/musiccmt.txt misc/musiccmtt.txt wav/se/*.wav wav/bgm/*.ogg wav/bgm/thbgm.fmt
var Assets embed.FS
