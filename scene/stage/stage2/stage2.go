// Package stage2 contains stage 2 configuration only.
package stage2

import (
	"th10/audio"
	stagescript "th10/scene/stage/script"
)

type Script2 = stagescript.Script

var Config = stagescript.Config{
	ECL:              "ecl/stage02.ecl",
	MSG:              "stage2",
	BGM:              audio.BGMStage2,
	BossBGM:          audio.BGMStage2Boss,
	FrameCap:         60000,
	BackgroundStage:  2,
	BackgroundScroll: 2.0,
	Backgrounds: [5]string{
		"anm/background/stg2bg.png",
		"anm/background/stg2bg2.png",
	},
}

var NewScript = Config.NewScript
