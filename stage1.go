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
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

var (
	imageWindows       = ebiten.NewImage(ScreenWidth, ScreenHeight)
	imagePanelBG, _, _ = ebitenutil.NewImageFromFileSystem(assets, "assets/img/stagebg.png")
	imageBackground    [4]*ebiten.Image
)

func init() {
	imageBackground[0], _, _ = ebitenutil.NewImageFromFileSystem(assets, "assets/img/stg3bg.png")
	imageBackground[1], _, _ = ebitenutil.NewImageFromFileSystem(assets, "assets/img/stg3bg2.png")
	imageBackground[2], _, _ = ebitenutil.NewImageFromFileSystem(assets, "assets/img/stg3bg3.png")
	imageBackground[3], _, _ = ebitenutil.NewImageFromFileSystem(assets, "assets/img/stg3bg4.png")
	// Windows: Field

	drawWindow(imageWindows, 32, 16, 960, 1120)

}

func drawWindow(r *ebiten.Image, x, y, width, height int) {
	ebitenutil.DrawRect(r, float64(x), float64(y), float64(width), float64(height), color.RGBA{0, 0, 0, 0xc0})
}

type Stage1 struct {
	player             player
	field              *Field
	currentPiece       *Piece
	currentPieceX      int
	currentPieceY      int
	currentPieceYCarry int
	currentPieceAngle  Angle
	nextPiece          *Piece
	landingCount       int
	score              int
	lines              int
	gameover           bool
	count              int
}

type player struct {
	slow       bool
	attack     bool
	bomb       bool
	miss       bool
	moveStatus int //0???????????????1?????? 2???3?????? 4???5??????
	x          float64
	y          float64
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

func NewStage1() *Stage1 {
	return &Stage1{
		field: &Field{},
	}
}

func (s *Stage1) choosePiece() *Piece {
	num := int(BlockTypeMax)
	blockType := BlockType(rand.Intn(num) + 1)
	return Pieces[blockType]
}

func (s *Stage1) initCurrentPiece(piece *Piece) {
	s.currentPiece = piece
	s.currentPieceYCarry = 0
	s.currentPieceAngle = Angle0
}

func (s *Stage1) level() int {
	return s.lines / 10
}

func (s *Stage1) addScore(lines int) {
	base := 0
	switch lines {
	case 1:
		base = 100
	case 2:
		base = 300
	case 3:
		base = 600
	case 4:
		base = 1000
	default:
		panic("not reach")
	}
	s.score += (s.level() + 1) * base
}

func (s *Stage1) Update(state *GameState) error {
	s.count++

	if ebiten.IsKeyPressed(ebiten.KeyZ) {
		s.player.attack = true
		playsound("assets/img/se_tan01.wav")
		return nil
	}

	if s.count%20 == 0 {
		s.score += rand.Intn(10000)
	}

	if ebiten.IsKeyPressed(ebiten.KeyArrowUp) {
		if ebiten.IsKeyPressed(ebiten.KeyShiftLeft) {
			s.player.y -= 2
		} else {
			s.player.y -= 6
		}
	} else if ebiten.IsKeyPressed(ebiten.KeyArrowDown) {
		if ebiten.IsKeyPressed(ebiten.KeyShiftLeft) {
			s.player.y += 2
		} else {
			s.player.y += 6
		}
	}

	if ebiten.IsKeyPressed(ebiten.KeyArrowRight) {
		s.player.moveStatus = 2
		if ebiten.IsKeyPressed(ebiten.KeyShiftLeft) {
			s.player.x += 2
		} else {
			s.player.x += 6
		}
	} else if ebiten.IsKeyPressed(ebiten.KeyArrowLeft) {
		s.player.moveStatus = 1
		if ebiten.IsKeyPressed(ebiten.KeyShiftLeft) {
			s.player.x -= 2
		} else {
			s.player.x -= 6
		}
	} else {
		s.player.moveStatus = 0
	}
	return nil
}

func (s *Stage1) Draw(r *ebiten.Image) {
	s.drawGameBg(r, imageBackground[2], s.count)

	s.drawPlayer(r, s.count)
	s.layoutInfo(r)
}

func (s *Stage1) drawPlayer(r *ebiten.Image, c int) {
	m1, _, _ := ebitenutil.NewImageFromFileSystem(assets, "assets/img/pl00.png")
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(s.player.x, s.player.y)
	i := (s.count / 5) % 8
	sx, sy := i*64, s.player.moveStatus*96
	r.DrawImage(m1.SubImage(image.Rect(sx, sy, sx+64, sy+96)).(*ebiten.Image), op)
}

func (s *Stage1) drawGameBg(r *ebiten.Image, img *ebiten.Image, c int) {
	w, h := img.Size()
	op := &ebiten.DrawImageOptions{}
	for i := 0; i < (ScreenWidth/w+1)*(ScreenHeight/h+2); i++ {
		op.GeoM.Reset()
		dy := (c) % h
		dstY := (i/(ScreenWidth/w+1)-1)*h + dy
		op.GeoM.Translate(80, float64(dstY))
		//fmt.Println(op.GeoM)
		//????????????
		//lineW := w + i*3/4
		//x := -float64(lineW) / float64(w) / 2
		//op.GeoM.Scale(float64(lineW)/float64(w), 1)
		//op.GeoM.Translate(x, float64(i))
		r.DrawImage(img, op)
	}
}
func (s *Stage1) layoutInfo(r *ebiten.Image) {
	//????????????
	r.DrawImage(imagePanelBG, nil)
	//info??????
	drawTextWithShadow(r, "Hiscore\n\n\nScore\n\n\nPlayer\n\n\nPower", 0.7*ScreenWidth, 0.05*ScreenHeight, 4, color.White)
	//??????
	drawTextWithShadow(r, fmt.Sprintf("%09d\n\n\n%09d", s.score, s.score), 0.8*ScreenWidth, 0.05*ScreenHeight, 4, color.White)
	//??????
	ebitenutil.DebugPrintAt(r, fmt.Sprintf("%0.2f fps", ebiten.ActualTPS()), ScreenWidth-70, ScreenHeight-20)
}
