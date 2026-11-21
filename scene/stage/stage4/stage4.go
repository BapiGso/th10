// Package stage4 contains stage 4 configuration only.
package stage4

import (
	"th10/audio"
	stagescript "th10/scene/stage/script"
)

type Script4 = stagescript.Script

var Config = stagescript.Config{
	ECL:              "ecl/stage04.ecl",
	MSG:              "stage4",
	BGM:              audio.BGMStage4,
	BossBGM:          audio.BGMStage4Boss,
	FrameCap:         60000,
	BackgroundStage:  4,
	BackgroundScroll: 2.5,
	Backgrounds: [5]string{
		"anm/background/stg4bg.png",
		"anm/background/stg4bg3.png",
		"anm/background/stg4bg7.png",
	},
}

var NewScript = Config.NewScript
