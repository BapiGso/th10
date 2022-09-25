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
	"image/color"
	"log"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
)

const (
	arcadeFontBaseSize = 8
)

var (
	arcadeFonts map[int]font.Face
	ttf, _      = assets.ReadFile("assets/ttf/MadokaLetters.ttf")
)

// 解析使用字体文件并且指定font size
func getArcadeFonts(scale int) font.Face {
	if arcadeFonts == nil {
		tt, err := opentype.Parse(ttf)
		if err != nil {
			log.Fatal(err)
		}
		arcadeFonts = map[int]font.Face{}
		for i := 1; i <= 4; i++ {
			arcadeFonts[i], err = opentype.NewFace(tt, &opentype.FaceOptions{
				Size:    float64(arcadeFontBaseSize * i),
				DPI:     72,
				Hinting: font.HintingFull,
			})
			if err != nil {
				log.Fatal(err)
			}
		}
	}
	return arcadeFonts[scale]
}

func textWidth(str string) int {
	maxW := 0
	for _, line := range strings.Split(str, "\n") {
		b, _ := font.BoundString(getArcadeFonts(1), line)
		w := (b.Max.X - b.Min.X).Ceil()
		if maxW < w {
			maxW = w
		}
	}
	return maxW
}

var (
	shadowColor = color.NRGBA{0, 0, 0, 255}
)

func drawTextWithShadow(rt *ebiten.Image, str string, x, y, scale int, clr color.Color) {
	op := &ebiten.DrawImageOptions{}
	offsetY := arcadeFontBaseSize * scale
	for _, line := range strings.Split(str, "\n") {
		y += offsetY
		op.GeoM.Scale(1.1, 1.1)
		op.GeoM.Translate(float64(x), float64(y))
		//
		//op.ColorM.ScaleWithColor(color.Black)
		//text.DrawWithOptions(rt, line, getArcadeFonts(scale), op)
		//op.GeoM.Scale(1, 1)
		//op.ColorM.ScaleWithColor(color.White)
		//text.DrawWithOptions(rt, line, getArcadeFonts(scale), op)
		text.Draw(rt, line, getArcadeFonts(scale), x, y, clr)
	}
}

func drawTextWithShadowCenter(rt *ebiten.Image, str string, x, y, scale int, clr color.Color, width int) {
	w := textWidth(str) * scale
	x += (width - w) / 2
	drawTextWithShadow(rt, str, x, y, scale, clr)
}

func drawTextWithShadowRight(rt *ebiten.Image, str string, x, y, scale int, clr color.Color, width int) {
	w := textWidth(str) * scale
	x += width - w
	drawTextWithShadow(rt, str, x, y, scale, clr)
}
