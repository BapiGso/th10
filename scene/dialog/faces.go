package dialog

import (
	"bytes"
	"image"
	_ "image/png"
	"sync"
	"th10/assets"

	"github.com/hajimehoshi/ebiten/v2"
)

// speakerFacePath maps speaker name to embedded asset path (normal expression, upper half).
var speakerFacePath = map[string]string{
	// Player
	"灵梦":     "anm/face/pl00/face_pl00no_u.png",
	"博麗霊夢":   "anm/face/pl00/face_pl00no_u.png",
	"魔理沙":    "anm/face/pl01/face_pl01no_u.png",
	"霧雨魔理沙":  "anm/face/pl01/face_pl01no_u.png",
	// Stage 1 — 秋姉妹
	"秋静葉": "anm/face/enemy1/face01no_u.png",
	"秋穣子": "anm/face/enemy1/face01no_u.png",
	// Stage 2
	"鍵山雛": "anm/face/enemy2/face02no_u.png",
	// Stage 3
	"河城にとり": "anm/face/enemy3/face03no_u.png",
	// Stage 4
	"射命丸文": "anm/face/enemy4/face04no_u.png",
	"犬走椛":  "anm/face/enemy4/face04no_u.png",
	// Stage 5
	"東風谷早苗": "anm/face/enemy5/face05no_u.png",
	// Stage 6
	"八坂神奈子": "anm/face/enemy6/face06no_u.png",
	// Extra
	"洩矢諏訪子": "anm/face/enemy7/face07no_u.png",
}

var (
	faceCache     map[string]*ebiten.Image
	faceCacheOnce sync.Once
)

// loadFaces lazily loads all face images into faceCache (called once via sync.Once).
func loadFaces() {
	faceCacheOnce.Do(func() {
		faceCache = make(map[string]*ebiten.Image, len(speakerFacePath))
		// Deduplicate paths — multiple speaker names may share the same image file.
		loaded := make(map[string]*ebiten.Image)
		for speaker, path := range speakerFacePath {
			if img, ok := loaded[path]; ok {
				faceCache[speaker] = img
				continue
			}
			img := loadFaceImage(path)
			loaded[path] = img
			faceCache[speaker] = img
		}
	})
}

// loadFaceImage reads a PNG from the embedded FS and returns an *ebiten.Image.
func loadFaceImage(path string) *ebiten.Image {
	data, err := assets.Assets.ReadFile(path)
	if err != nil {
		return nil
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil
	}
	return ebiten.NewImageFromImage(img)
}

// getFace returns the face image for the given speaker, or nil if none.
func getFace(speaker string) *ebiten.Image {
	loadFaces()
	return faceCache[speaker]
}
