package title

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/joelschutz/stagehand"
)

type Resource interface {
	init()
	Update() error
	Render() (*ebiten.Image, *ebiten.DrawImageOptions) //子img的draw action并返回用于主渲染
}

type Title struct {
	count     int
	resources []Resource
	*stagehand.SceneManager[any]
}

func (t *Title) Load(test any, sm stagehand.SceneController[any]) {
	t.resources = append(t.resources, new(sig)) //new(bgm))
	for _, resource := range t.resources {
		resource.init()
	}
	t.SceneManager = sm.(*stagehand.SceneManager[any])
}

func (t *Title) Unload() any {
	return 0
}
func (t *Title) Update() error {
	t.count++
	return nil
}

func (t *Title) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return outsideWidth, outsideHeight
}

func (t *Title) Draw(screen *ebiten.Image) {
	for _, resource := range t.resources {
		screen.DrawImage(resource.Render())
	}
}
