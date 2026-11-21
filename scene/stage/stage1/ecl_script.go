// Package stage1 contains stage 1 configuration only.
package stage1

import (
	"th10/audio"
	stagescript "th10/scene/stage/script"
)

var Config = stagescript.Config{
	ECL:              "ecl/stage01.ecl",
	MSG:              "stage1",
	BGM:              audio.BGMStage1,
	BossBGM:          audio.BGMStage1Boss,
	FrameCap:         60000,
	BackgroundStage:  1,
	BackgroundScroll: 0,
	Backgrounds: [5]string{
		"anm/background/stg1bg.png",
		"anm/background/stg1bg2.png",
		"anm/background/stg1bg3.png",
		"anm/background/stg1bg4.png",
	},
}

var NewScript = Config.NewScriptInterface
var NewECLScript = NewScript
