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
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"math/rand"
	"strconv"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

var (
	imageGameBG           *ebiten.Image
	imageWindows          = ebiten.NewImage(ScreenWidth, ScreenHeight)
	imageBackground, _, _ = ebitenutil.NewImageFromFileSystem(assets, "assets/stg1bg.png")
)

func fieldWindowPosition() (x, y int) {
	return 200, 200
}

func nextWindowLabelPosition() (x, y int) {
	x, y = fieldWindowPosition()
	return x + fieldWidth + 2*blockWidth, y
}

func nextWindowPosition() (x, y int) {
	x, y = nextWindowLabelPosition()
	return x, y + blockHeight
}

func textBoxWidth() int {
	x, _ := nextWindowPosition()
	return ScreenWidth - 2*blockWidth - x
}

func scoreTextBoxPosition() (x, y int) {
	x, y = nextWindowPosition()
	return x, y + 6*blockHeight
}

func levelTextBoxPosition() (x, y int) {
	x, y = scoreTextBoxPosition()
	return x, y + 4*blockHeight
}

func linesTextBoxPosition() (x, y int) {
	x, y = levelTextBoxPosition()
	return x, y + 4*blockHeight
}

func init() {
	// Background
	img, _, err := ebitenutil.NewImageFromFileSystem(assets, "assets/gamebg.png")
	if err != nil {
		panic(err)
	}
	imageGameBG = ebiten.NewImageFromImage(img)

	// Windows: Field

	drawWindow(imageWindows, 32, 16, 384, 448)

	// Windows: Next

	drawTextBox(imageWindows, "HiScore", 430, 20, textBoxWidth())

	// Windows: Score

	drawTextBox(imageWindows, "SCORE", 430, 50, textBoxWidth())

	// Windows: Level

	drawTextBox(imageWindows, "Player", 430, 80, textBoxWidth())

	// Windows: Lines

	drawTextBox(imageWindows, "Power", 430, 110, textBoxWidth())

	// Gameover

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

type GameScene struct {
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

func init() {
	rand.Seed(time.Now().UnixNano())
}

func NewGameScene() *GameScene {
	return &GameScene{
		field: &Field{},
	}
}

func (s *GameScene) drawBackground(r *ebiten.Image) {
	r.Fill(color.White)

	w, h := imageGameBG.Size()
	scaleW := ScreenWidth / float64(w)
	scaleH := ScreenHeight / float64(h)
	scale := scaleW
	if scale < scaleH {
		scale = scaleH
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-float64(w)/2, -float64(h)/2)
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(ScreenWidth/2, ScreenHeight/2)
	op.Filter = ebiten.FilterLinear
	r.DrawImage(imageGameBG, op)
}

const (
	fieldWidth  = blockWidth * fieldBlockCountX
	fieldHeight = blockHeight * fieldBlockCountY
)

func (s *GameScene) choosePiece() *Piece {
	num := int(BlockTypeMax)
	blockType := BlockType(rand.Intn(num) + 1)
	return Pieces[blockType]
}

func (s *GameScene) initCurrentPiece(piece *Piece) {
	s.currentPiece = piece
	s.currentPieceYCarry = 0
	s.currentPieceAngle = Angle0
}

func (s *GameScene) level() int {
	return s.lines / 10
}

func (s *GameScene) addScore(lines int) {
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

func (s *GameScene) Update(state *GameState) error {
	s.count++
	if ebiten.IsKeyPressed(ebiten.KeyX) {
		playsound("assets/se_tan01.wav")
		return nil
	}
	return nil
	s.field.Update()

	if s.gameover {
		if inpututil.IsKeyJustPressed(ebiten.KeySpace) ||
			anyGamepadVirtualButtonJustPressed(state.Input) {
			state.SceneManager.GoTo(&WelcomeScene{})
		}
		return nil
	}

	maxLandingCount := ebiten.TPS()
	if s.currentPiece == nil {
		s.initCurrentPiece(s.choosePiece())
	}
	if s.nextPiece == nil {
		s.nextPiece = s.choosePiece()
	}

	moved := false
	piece := s.currentPiece
	angle := s.currentPieceAngle

	// Move piece by user input.
	if !s.field.IsFlushAnimating() {
		piece := s.currentPiece
		x := s.currentPieceX
		y := s.currentPieceY
		if state.Input.IsRotateRightJustPressed() {
			s.currentPieceAngle = s.field.RotatePieceRight(piece, x, y, angle)
			moved = angle != s.currentPieceAngle
		} else if state.Input.IsRotateLeftJustPressed() {
			s.currentPieceAngle = s.field.RotatePieceLeft(piece, x, y, angle)
			moved = angle != s.currentPieceAngle
		} else if l := state.Input.StateForLeft(); l == 1 || (10 <= l && l%2 == 0) {
			s.currentPieceX = s.field.MovePieceToLeft(piece, x, y, angle)
			moved = x != s.currentPieceX
		} else if r := state.Input.StateForRight(); r == 1 || (10 <= r && r%2 == 0) {
			s.currentPieceX = s.field.MovePieceToRight(piece, x, y, angle)
			moved = y != s.currentPieceX
		} else if d := state.Input.StateForDown(); (d-1)%2 == 0 {
			s.currentPieceY = s.field.DropPiece(piece, x, y, angle)
			moved = y != s.currentPieceY
			if moved {
				s.score++
			}
		}
	}

	// Drop the current piece with gravity.
	if !s.field.IsFlushAnimating() {
		angle := s.currentPieceAngle
		s.currentPieceYCarry += 2*s.level() + 1
		const maxCarry = 60
		for maxCarry <= s.currentPieceYCarry {
			s.currentPieceYCarry -= maxCarry
			s.currentPieceY = s.field.DropPiece(piece, s.currentPieceX, s.currentPieceY, angle)
		}
	}

	if !s.field.IsFlushAnimating() && !s.field.PieceDroppable(piece, s.currentPieceX, s.currentPieceY, angle) {
		if 0 < state.Input.StateForDown() {
			s.landingCount += 10
		} else {
			s.landingCount++
		}
		if maxLandingCount <= s.landingCount {
			s.field.AbsorbPiece(piece, s.currentPieceX, s.currentPieceY, angle)
			if s.field.IsFlushAnimating() {
				s.field.SetEndFlushAnimating(func(lines int) {
					s.lines += lines
					if 0 < lines {
						s.addScore(lines)
					}
					s.goNextPiece()
				})
			} else {
				s.goNextPiece()
			}

		}
	}
	return nil
}

func (s *GameScene) goNextPiece() {
	s.initCurrentPiece(s.nextPiece)
	s.nextPiece = s.choosePiece()
	s.landingCount = 0
	if s.currentPiece.collides(s.field, s.currentPieceX, s.currentPieceY, s.currentPieceAngle) {
		s.gameover = true
	}
}

func (s *GameScene) Draw(r *ebiten.Image) {
	s.drawBackground(r)
	msg := fmt.Sprintf("%0.2f fps", ebiten.ActualTPS())
	ebitenutil.DebugPrintAt(r, msg, ScreenWidth-70, ScreenHeight-20)
	//r.DrawImage(imageWindows, nil)

	// Draw score
	x, y := scoreTextBoxPosition()
	drawTextBoxContent(r, strconv.Itoa(s.score), x, y, textBoxWidth())

	// Draw level
	x, y = levelTextBoxPosition()
	drawTextBoxContent(r, strconv.Itoa(s.level()), x, y, textBoxWidth())

	// Draw lines
	x, y = linesTextBoxPosition()
	s.drawTitleBackground(r, s.count)
}

func (s *GameScene) drawTitleBackground(r *ebiten.Image, c int) {
	w, h := imageBackground.Size()
	op := &ebiten.DrawImageOptions{}
	for i := 0; i < 200; i++ {
		op.GeoM.Reset()
		dy := (c) % h
		dstY := (i/(ScreenWidth/w+1)-1)*h + dy
		op.GeoM.Translate(30, float64(dstY)*10)
		r.DrawImage(imageBackground, op)
	}
}
