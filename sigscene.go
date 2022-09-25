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
)

var ImgCache = &Img{}

func init() {
	ImgCache.Sig, _, _ = ebitenutil.NewImageFromFileSystem(assets, "assets/img/sig.png")
	ImgCache.Loading, _, _ = ebitenutil.NewImageFromFileSystem(assets, "assets/img/loading.png")
}

type Img struct {
	Sig     *ebiten.Image   //签名图
	TitleBg *ebiten.Image   //标题的背景图
	Logo    []*ebiten.Image //标题的logo
	Loading *ebiten.Image   //少女祈祷中。。。
	Menu    []*ebiten.Image //菜单
	Player  []*ebiten.Image //选人
	Skill   []*ebiten.Image //技能
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
	if s.count == 100 {
		go bgmsw("assets/wav/th10_01.mp3")
		state.SceneManager.GoTo(&WelcomeScene{})
		return nil
	}
	return nil
}

func (s *SigScene) Draw(r *ebiten.Image) {
	op := &ebiten.DrawImageOptions{}
	r.DrawImage(ImgCache.Sig, op)
	op.GeoM.Translate((ScreenWidth - 180), (ScreenHeight - 100))
	r.DrawImage(ImgCache.Loading, op)
}
