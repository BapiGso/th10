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
	imagePanelBG, _, _ = ebitenutil.NewImageFromFileSystem(assets, "assets/img/gamebg.png")
	imageBackground    [4]*ebiten.Image
)

func init() {
	imageBackground[0], _, _ = ebitenutil.NewImageFromFileSystem(assets, "assets/img/stg3bg.png")
	imageBackground[1], _, _ = ebitenutil.NewImageFromFileSystem(assets, "assets/img/stg3bg2.png")
	imageBackground[2], _, _ = ebitenutil.NewImageFromFileSystem(assets, "assets/img/stg3bg3.png")
	imageBackground[3], _, _ = ebitenutil.NewImageFromFileSystem(assets, "assets/img/stg3bg4.png")
	NewStage1().player.x, NewStage1().player.y = 300, 400
	// Windows: Field

	drawWindow(imageWindows, 32, 16, 384, 448)

}

func (s *Stage1) init() {
	s.player.x, s.player.y = 500, 300
}

func joinimg() {
	_, m, _ := ebitenutil.NewImageFromFileSystem(assets, "assets/img/pl00.png")
	for i := 0; i < 8; i++ {
		if m == nil {

		}
	}
}

func subimg(img image.Image, x0, y0, x1, y1 int) *image.NRGBA {
	subImage := img.(*image.NRGBA).SubImage(image.Rect(x0, y0, x1, y1)).(*image.NRGBA)
	return subImage
}

func drawWindow(r *ebiten.Image, x, y, width, height int) {
	ebitenutil.DrawRect(r, float64(x), float64(y), float64(width), float64(height), color.RGBA{0, 0, 0, 0xc0})
}

var fontColor = color.NRGBA{0x40, 0x40, 0xff, 0xff}

func drawTextBox(r *ebiten.Image, label string, x, y, width int) {
	drawTextWithShadow(r, label, x, y, 1, fontColor)
	y += blockWidth
	drawWindow(r, x, y, width, 2*blockHeight)
}

func drawTextBoxContent(r *ebiten.Image, content string, x, y, width int) {
	y += blockWidth
	drawTextWithShadowRight(r, content, x, y+blockHeight*3/4, 1, color.White, width-blockWidth/2)
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
	moveStatus int //0是没转向，1低速 2左3左低 4右5右低
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

const (
	fieldWidth  = blockWidth * fieldBlockCountX
	fieldHeight = blockHeight * fieldBlockCountY
)

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

	if ebiten.IsKeyPressed(ebiten.KeyArrowUp) {
		if ebiten.IsKeyPressed(ebiten.KeyShiftLeft) {
			s.player.y -= 1
		} else {
			s.player.y -= 3
		}
	} else if ebiten.IsKeyPressed(ebiten.KeyArrowDown) {
		if ebiten.IsKeyPressed(ebiten.KeyShiftLeft) {
			s.player.y += 1
		} else {
			s.player.y += 3
		}
	}

	if ebiten.IsKeyPressed(ebiten.KeyArrowRight) {
		s.player.moveStatus = 2
		if ebiten.IsKeyPressed(ebiten.KeyShiftLeft) {
			s.player.x += 1
		} else {
			s.player.x += 3
		}
	} else if ebiten.IsKeyPressed(ebiten.KeyArrowLeft) {
		s.player.moveStatus = 1
		if ebiten.IsKeyPressed(ebiten.KeyShiftLeft) {
			s.player.x -= 1
		} else {
			s.player.x -= 3
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
	sx, sy := i*32, s.player.moveStatus*48
	r.DrawImage(m1.SubImage(image.Rect(sx, sy, sx+32, sy+48)).(*ebiten.Image), op)
}

func (s *Stage1) drawGameBg(r *ebiten.Image, img *ebiten.Image, c int) {
	w, h := img.Size()
	op := &ebiten.DrawImageOptions{}
	for i := 0; i < (ScreenWidth/w+1)*(ScreenHeight/h+2); i++ {
		op.GeoM.Reset()
		dy := (c) % h
		dstY := (i/(ScreenWidth/w+1)-1)*h + dy
		op.GeoM.Translate(32, float64(dstY))
		//fmt.Println(op.GeoM)
		//透视矩阵
		//lineW := w + i*3/4
		//x := -float64(lineW) / float64(w) / 2
		//op.GeoM.Scale(float64(lineW)/float64(w), 1)
		//op.GeoM.Translate(x, float64(i))
		r.DrawImage(img, op)
	}
}
func (s *Stage1) layoutInfo(r *ebiten.Image) {

	//面板背景
	r.DrawImage(imagePanelBG, nil)
	//info面板
	drawTextWithShadow(r, "Hiscore\n\n\nScore\n\n\nPlayer\n\n\nPower", 430, 15, 2, color.White)
	//帧数
	ebitenutil.DebugPrintAt(r, fmt.Sprintf("%0.2f fps", ebiten.ActualTPS()), ScreenWidth-70, ScreenHeight-20)
}
