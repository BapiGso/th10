// Package stage6 contains stage 6 configuration only.
package stage6

import (
	"th10/audio"
	stagescript "th10/scene/stage/script"
)

type Script6 = stagescript.Script

var Config = stagescript.Config{
	ECL:              "ecl/stage06.ecl",
	MSG:              "stage6",
	BGM:              audio.BGMStage6,
	BossBGM:          audio.BGMStage6Boss,
	FrameCap:         60000,
	BackgroundStage:  6,
	BackgroundScroll: 1.2,
	Backgrounds: [5]string{
		"anm/background/stg6bg.png",
		"anm/background/stg6bg2.png",
		"anm/background/stg6bg3.png",
		"anm/background/stg6bg5.png",
		"anm/background/stg6bg6.png",
	},
}

var NewScript = Config.NewScript
