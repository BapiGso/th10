// Package extra contains stage 7 configuration only.
package extra

import (
	"th10/audio"
	stagescript "th10/scene/stage/script"
)

type ScriptExtra = stagescript.Script

var Config = stagescript.Config{
	ECL:              "ecl/stage07.ecl",
	MSG:              "extra",
	BGM:              audio.BGMExtra,
	BossBGM:          audio.BGMExtraBoss,
	FrameCap:         60000,
	BackgroundStage:  7,
	BackgroundScroll: 2.0,
	Backgrounds: [5]string{
		"anm/background/stg7bg.png",
		"anm/background/stg7bg2.png",
		"anm/background/stg7bg3.png",
	},
}

var NewScript = Config.NewScript
