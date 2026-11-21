// Package stage5 contains stage 5 configuration only.
package stage5

import (
	"th10/audio"
	stagescript "th10/scene/stage/script"
)

type Script5 = stagescript.Script

var Config = stagescript.Config{
	ECL:              "ecl/stage05.ecl",
	MSG:              "stage5",
	BGM:              audio.BGMStage5,
	BossBGM:          audio.BGMStage5Boss,
	FrameCap:         60000,
	BackgroundStage:  5,
	BackgroundScroll: 1.5,
	Backgrounds: [5]string{
		"anm/background/stg5bg.png",
		"anm/background/stg5bg2.png",
	},
}

var NewScript = Config.NewScript
