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
	"bytes"
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/mp3"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"image"
	"image/color"
	_ "image/png"
)

type WelcomeScene struct {
	count  int //时间t
	status int //菜单状态
}

var Welcomeimg [10]*ebiten.Image
var audioContext = audio.NewContext(SampleRate)
var bgm *audio.Player

func init() {
	Welcomeimg[0], _, _ = ebitenutil.NewImageFromFileSystem(assets, "assets/img/title00a.png")
	_, m, _ := ebitenutil.NewImageFromFileSystem(assets, "assets/img/title_logo.png")
	_, m1, _ := ebitenutil.NewImageFromFileSystem(assets, "assets/img/title01.png")
	subImage1 := m.(*image.NRGBA).SubImage(image.Rect(0, 0, 512, 128)).(*image.NRGBA)
	subImage2 := m.(*image.NRGBA).SubImage(image.Rect(0, 128, 512, 256)).(*image.NRGBA)
	subImage3 := m1.(*image.NRGBA).SubImage(image.Rect(0, 0, 160, 30)).(*image.NRGBA)
	subImage4 := m1.(*image.NRGBA).SubImage(image.Rect(0, 30, 161, 60)).(*image.NRGBA)
	subImage5 := m1.(*image.NRGBA).SubImage(image.Rect(0, 60, 208, 96)).(*image.NRGBA)
	subImage6 := m1.(*image.NRGBA).SubImage(image.Rect(0, 90, 96, 128)).(*image.NRGBA)
	Welcomeimg[1] = ebiten.NewImageFromImage(subImage1)
	Welcomeimg[2] = ebiten.NewImageFromImage(subImage2)
	Welcomeimg[3] = ebiten.NewImageFromImage(subImage3)
	Welcomeimg[4] = ebiten.NewImageFromImage(subImage4)
	Welcomeimg[5] = ebiten.NewImageFromImage(subImage5)
	Welcomeimg[6] = ebiten.NewImageFromImage(subImage6)
}

// 停止当前播放的bgm并播放新的bgm
func bgmsw(path string) {
	if bgm != nil && bgm.IsPlaying() {
		bgm.Pause()
	}
	Welcomeimgm, _ := assets.ReadFile(path)
	s, _ := mp3.DecodeWithoutResampling(bytes.NewReader(Welcomeimgm))
	bgm, _ = audioContext.NewPlayer(s)
	bgm.Play()
}

// 播放一个音效
func playsound(path string) {
	soundfile, _ := assets.ReadFile(path)
	s, _ := wav.DecodeWithoutResampling(bytes.NewReader(soundfile))
	sound, _ := audioContext.NewPlayer(s)
	sound.Play()
}

func (s *WelcomeScene) Update(state *GameState) error {
	s.count++

	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		s.status = 1
		return nil
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyZ) {
		bgmsw("assets/wav/th10_02.mp3")
		//return nil
		state.SceneManager.GoTo(NewStage1())
		return nil
	}

	if ebiten.IsKeyPressed(ebiten.KeyX) {
		playsound("assets/wav/se_tan01.wav")
		return nil
	}

	return nil
}

func (s *WelcomeScene) Draw(r *ebiten.Image) {
	s.drawImg(r, Welcomeimg[0], 0, 0, s.count)
	//drawLogo(r, "BLOCKS")
	ebitenutil.DebugPrintAt(r, fmt.Sprintf("%0.2f fps", ebiten.ActualTPS()), ScreenWidth-70, ScreenHeight-20)
	if s.status == 0 {
		s.drawImg(r, Welcomeimg[1], (ScreenWidth-512)/2, (ScreenHeight-128)/2, s.count)
		message := "PRESS ANY BUTTON"
		x := 0
		y := ScreenHeight - 100
		drawTextWithShadowCenter(r, message, x, y, 3, color.NRGBA{255, 255, 255, 255}, ScreenWidth)
	} else if s.status == 1 {
		s.drawImg(r, Welcomeimg[2], ScreenWidth*0.05, ScreenHeight*0.4, s.count)
		s.drawImg(r, Welcomeimg[3], ScreenWidth-240, ScreenHeight-330, s.count)
		s.drawImg(r, Welcomeimg[4], ScreenWidth-230, ScreenHeight-300, s.count)
		s.drawImg(r, Welcomeimg[5], ScreenWidth-220, ScreenHeight-270, s.count)
		s.drawImg(r, Welcomeimg[6], ScreenWidth-210, ScreenHeight-240, s.count)
	}

}

func (s *WelcomeScene) drawImg(r *ebiten.Image, img *ebiten.Image, x, y float64, c int) {
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
	op.GeoM.Translate(x, y)
	r.DrawImage(img, op)
}
