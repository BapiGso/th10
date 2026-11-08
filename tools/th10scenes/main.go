// Command th10scenes is a development-only viewer for scene-B fidelity
// checks. It uses embedded assets and the normal game/audio/store APIs;
// it never reads an original install or changes the scene supervisor.
package main

import (
	"flag"
	"log"

	"th10/game"
	"th10/scene/ending"
	"th10/scene/menus"
	"th10/scene/title"

	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	scene := flag.String("scene", "ending", "ending, staff, music, data, or replay")
	character := flag.Int("character", 0, "0 Reimu, 1 Marisa")
	shot := flag.Int("shot", 0, "0 A, 1 B, 2 C")
	difficulty := flag.Int("difficulty", 1, "0 Easy, 1 Normal, 2 Hard, 3 Lunatic")
	continues := flag.Int("continues", 0, "original continuation count (0..9)")
	seen := flag.Bool("seen", false, "mark ending/Staff seen to exercise Ctrl skip")
	flag.Parse()
	if *character < 0 || *character > 1 || *shot < 0 || *shot > 2 || *difficulty < 0 || *difficulty > 3 || *continues < 0 || *continues > 9 {
		log.Fatal("invalid character, shot, difficulty, or continuation count")
	}
	g := game.New()
	state := g.NewState(*character, *shot, *difficulty)
	state.Stage, state.ReplayState90 = 6, int32(*continues)
	back := title.New()
	var next game.Scene
	switch *scene {
	case "ending":
		if *seen {
			index, err := ending.EndingIndex(state)
			if err != nil {
				log.Fatal(err)
			}
			g.Save.Native.MarkEndingSeen(index)
		}
		e := ending.New(state, false, back)
		e.After = func(*game.Game) game.Scene { return back } // viewer only, not the Stage's ranking flow
		next = e
	case "staff":
		if *seen {
			g.Save.Native.MarkStaffSeen()
		}
		s := ending.NewStaff(state, back)
		s.After = func(*game.Game) game.Scene { return back }
		next = s
	case "music":
		g.Audio().PlayBGM(0)
		next = menus.NewMusicRoom(back)
	case "data":
		g.Audio().PlayBGM(0)
		next = menus.NewPlayerData(g.Save, back)
	case "replay":
		g.Audio().PlayBGM(0)
		next = menus.NewReplay(back)
	default:
		log.Fatal("unknown scene")
	}
	ebiten.SetWindowSize(game.ScreenWidth, game.ScreenHeight)
	ebiten.SetWindowTitle("TH10 - Scene B Fidelity Viewer")
	g.SetScene(next)
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
