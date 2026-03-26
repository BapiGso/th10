// Package assets 这个目录里的东西不要只能增不能删
package assets

import "embed"

//go:embed anm font
//go:embed wav/se_bonus3.wav wav/se_cancel00.wav wav/se_cardget.wav wav/se_cat00.wav
//go:embed wav/se_ch00.wav wav/se_ch01.wav wav/se_damage00.wav wav/se_damage01.wav
//go:embed wav/se_enep00.wav wav/se_enep01.wav wav/se_extend.wav wav/se_graze.wav
//go:embed wav/se_gun00.wav wav/se_hint00.wav wav/se_invalid.wav wav/se_item00.wav
//go:embed wav/se_kira00.wav wav/se_kira01.wav wav/se_kira02.wav wav/se_lazer00.wav
//go:embed wav/se_lazer01.wav wav/se_ok00.wav wav/se_option.wav wav/se_pause.wav
//go:embed wav/se_pldead00.wav wav/se_plst00.wav wav/se_power0.wav wav/se_power1.wav
//go:embed wav/se_powerup.wav wav/se_select00.wav wav/se_slash.wav wav/se_tan00.wav
//go:embed wav/se_tan01.wav wav/se_tan02.wav wav/se_timeout.wav wav/se_timeout2.wav
//go:embed wav/se_water.wav
//go:embed wav/bgm_ogg
var Assets embed.FS
