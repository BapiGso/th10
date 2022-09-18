// Copyright 2014 Hajime Hoshi
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	_ "image/png"
	"time"
)

var Sigbg [5]*ebiten.Image
var Sigstart bool

func init() {
	Sigbg[0], _, _ = ebitenutil.NewImageFromFileSystem(assets, "assets/sig.png")
	Sigbg[1], _, _ = ebitenutil.NewImageFromFileSystem(assets, "assets/loading.png")
	go Signext()
}

func Signext() {
	time.Sleep(time.Second * 1)
	Sigstart = true
}

type SigScene struct {
	count int
}

func anyGamepadVirtualButtonJustPressed(i *Input) bool {
	if !i.gamepadConfig.IsGamepadIDInitialized() {
		return false
	}

	for _, b := range virtualGamepadButtons {
		if i.gamepadConfig.IsButtonJustPressed(b) {
			return true
		}
	}
	return false
}

func (s *SigScene) Update(state *GameState) error {
	s.count++
	if Sigstart {
		//return nil
		//time.Sleep(time.Second * 10)
		state.SceneManager.GoTo(&WelcomeScene{})
		return nil
	}
	return nil
}

func (s *SigScene) Draw(r *ebiten.Image) {
	s.drawSigBackground(r, s.count)
	//drawLogo(r, "BLOCKS")

}

func (s *SigScene) drawSigBackground(r *ebiten.Image, c int) {
	//w, h := imageBackground.Size()
	op := &ebiten.DrawImageOptions{}

	//for i := 0; i < (ScreenWidth/w+1)*(ScreenHeight/h+2); i++ {
	//	op.GeoM.Reset()
	//	dx := -(c / 1) % w
	//	dy := (c / 1) % h
	//	dstX := (i%(ScreenWidth/w+1))*w + dx
	//	dstY := (i/(ScreenWidth/w+1)-1)*h + dy
	//	op.GeoM.Translate(float64(dstX), float64(dstY))
	//	r.DrawImage(imageBackground, op)
	//}
	r.DrawImage(Sigbg[0], op)
	op.GeoM.Translate((ScreenWidth - 180), (ScreenHeight - 100))
	r.DrawImage(Sigbg[1], op)
}
