package stage

import (
	"fmt"

	"th10/game"
	"th10/gamedata"
	"th10/scene/dialog"
)

// attachMSG wires FUN_00415b00/00415e90 to the GUI's World and resources.
// Logic-only Contexts have no GUI and keep the existing interpreter alone.
func (g *GUI) attachMSG(m *dialog.MSG, stage, character int) {
	logo := g.logo
	if stage != g.ctx.State.Stage {
		logo = gamedata.MustANM(gamedata.SharedPresentationANM(fmt.Sprintf("st%02dlogo.anm", stage)))
	}
	m.SetPresentation(dialog.NewPresentation(dialog.PresentationConfig{
		Stage: stage, Character: character,
		World: g.ctx.World, Env: g.ctx.Anm,
		Front: g.front, Text: textFile(), Logo: logo,
		Player: gamedata.MustANM(gamedata.SharedANM(fmt.Sprintf("pl%02d.anm", character))),
		Enemy:  gamedata.MustANM(gamedata.SharedANM(fmt.Sprintf("stgenm%02d.anm", stage))),
		StageClear: func() {
			if g.ctx.State.RunFlags&game.RunFlagPractice == 0 { // FUN_00415e90 practice exits before banner
				g.ShowNotice(6, 0)
			}
		},
	}))
}
