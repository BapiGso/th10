package stage

import (
	"fmt"
	"sync"

	"th10/anm"
	"th10/game"
	"th10/gamedata"
	"th10/scene/dialog"
)

var messageANMs sync.Map

func messageANM(name string) *anm.File {
	if f, ok := messageANMs.Load(name); ok {
		return f.(*anm.File)
	}
	f, err := gamedata.ANM(name)
	if err != nil {
		panic("stage: MSG ANM: " + err.Error())
	}
	actual, _ := messageANMs.LoadOrStore(name, f)
	return actual.(*anm.File)
}

// attachMSG wires FUN_00415b00/00415e90 to the GUI's World and resources.
// Logic-only Contexts have no GUI and keep the existing interpreter alone.
func (g *GUI) attachMSG(m *dialog.MSG, stage, character int) {
	logo := g.logo
	if stage != g.ctx.State.Stage {
		f, err := gamedata.PresentationANM(fmt.Sprintf("st%02dlogo.anm", stage))
		if err != nil {
			panic("stage: MSG logo: " + err.Error())
		}
		logo = f
	}
	m.SetPresentation(dialog.NewPresentation(dialog.PresentationConfig{
		Stage: stage, Character: character,
		World: g.ctx.World, Env: g.ctx.Anm,
		Front: g.front, Text: textFile(), Logo: logo,
		Player: messageANM(fmt.Sprintf("pl%02d.anm", character)),
		Enemy:  messageANM(fmt.Sprintf("stgenm%02d.anm", stage)),
		StageClear: func() {
			if g.ctx.State.RunFlags&game.RunFlagPractice == 0 { // FUN_00415e90 practice exits before banner
				g.ShowNotice(6, 0)
			}
		},
	}))
}
