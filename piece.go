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
	"image"
	_ "image/png"

	"github.com/hajimehoshi/ebiten/v2"
	rblocks "github.com/hajimehoshi/ebiten/v2/examples/resources/images/blocks"
)

var imageBlocks *ebiten.Image

func init() {
	img, _, err := image.Decode(bytes.NewReader(rblocks.Blocks_png))
	if err != nil {
		panic(err)
	}
	imageBlocks = ebiten.NewImageFromImage(img)

}

type Angle int

const (
	Angle0 Angle = iota
	Angle90
	Angle180
	Angle270
)

func (a Angle) RotateRight() Angle {
	if a == Angle270 {
		return Angle0
	}
	return a + 1
}

func (a Angle) RotateLeft() Angle {
	if a == Angle0 {
		return Angle270
	}
	return a - 1
}

type BlockType int

const (
	BlockTypeNone BlockType = iota
	BlockType1
	BlockType2
	BlockType3
	BlockType4
	BlockType5
	BlockType6
	BlockType7
	BlockTypeMax = BlockType7
)

type Piece struct {
	blockType BlockType
	blocks    [][]bool
}

func transpose(bs [][]bool) [][]bool {
	blocks := make([][]bool, len(bs))
	for j, row := range bs {
		blocks[j] = make([]bool, len(row))
	}
	// Tranpose the argument matrix.
	for i, col := range bs {
		for j, v := range col {
			blocks[j][i] = v
		}
	}
	return blocks
}

// Pieces is the set of all the possible pieces.
var Pieces map[BlockType]*Piece

func init() {
	const (
		f = false
		t = true
	)
	Pieces = map[BlockType]*Piece{}
}

const (
	blockWidth       = 10
	blockHeight      = 10
	fieldBlockCountX = 10
	fieldBlockCountY = 20
)

// isBlocked returns a boolean value indicating whether
// there is a block at the position (x, y) of the piece with the given angle.
func (p *Piece) isBlocked(i, j int, angle Angle) bool {
	size := len(p.blocks)
	i2, j2 := i, j
	switch angle {
	case Angle0:
	case Angle90:
		i2 = j
		j2 = size - 1 - i
	case Angle180:
		i2 = size - 1 - i
		j2 = size - 1 - j
	case Angle270:
		i2 = size - 1 - j
		j2 = i
	}
	return p.blocks[i2][j2]
}

// collides returns a boolean value indicating whether
// the piece at (x, y) with the given angle would collide with the field's blocks.
func (p *Piece) collides(field *Field, x, y int, angle Angle) bool {
	size := len(p.blocks)
	for i := 0; i < size; i++ {
		for j := 0; j < size; j++ {
			if field.IsBlocked(x+i, y+j) && p.isBlocked(i, j, angle) {
				return true
			}
		}
	}
	return false
}

func (p *Piece) AbsorbInto(field *Field, x, y int, angle Angle) {
	size := len(p.blocks)
	for i := 0; i < size; i++ {
		for j := 0; j < size; j++ {
			if p.isBlocked(i, j, angle) {
				field.setBlock(x+i, y+j, p.blockType)
			}
		}
	}
}

func (p *Piece) DrawAtCenter(r *ebiten.Image, x, y, width, height int, angle Angle) {
	x += (width - len(p.blocks[0])*blockWidth) / 2
	y += (height - len(p.blocks)*blockHeight) / 2
	p.Draw(r, x, y, angle)
}

func (p *Piece) Draw(r *ebiten.Image, x, y int, angle Angle) {

}
