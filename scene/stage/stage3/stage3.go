// Package stage3 contains stage 3 configuration only.
package stage3

import (
	"th10/audio"
	stagescript "th10/scene/stage/script"
)

type Script3 = stagescript.Script

var Config = stagescript.Config{
	ECL:              "ecl/stage03.ecl",
	MSG:              "stage3",
	BGM:              audio.BGMStage3,
	BossBGM:          audio.BGMStage3Boss,
	FrameCap:         60000,
	BackgroundStage:  3,
	BackgroundScroll: 1.8,
	Backgrounds: [5]string{
		"anm/background/stg3bg.png",
		"anm/background/stg3bg2.png",
		"anm/background/stg3bg3.png",
		"anm/background/stg3bg4.png",
	},
}

var NewScript = Config.NewScript
