package stage

import (
	"testing"

	"th10/entity/item"
	"th10/game"
)

type panSoundRecord struct {
	ids  []int
	pans []struct {
		id int
		x  float32
	}
}

func (s *panSoundRecord) PlayBGM(int)     {}
func (s *panSoundRecord) StopBGM()        {}
func (s *panSoundRecord) FadeOutBGM(int)  {}
func (s *panSoundRecord) PlaySEID(id int) { s.ids = append(s.ids, id) }
func (s *panSoundRecord) PlaySEIDAt(id int, x float32) {
	s.pans = append(s.pans, struct {
		id int
		x  float32
	}{id, x})
}

func TestPositionalSoundWiring(t *testing.T) {
	c := NewContext(game.NewState(game.CharReimu, game.ShotA, game.DiffNormal))
	s := &panSoundRecord{}
	c.Audio = s
	c.BulletEnv.PlaySE(7, 96, true)
	c.LaserEnv.PlaySE(28, -96, true)
	c.BulletEnv.PlaySE(24, 96, false)
	if len(s.pans) != 2 || s.pans[0].id != 7 || s.pans[0].x != 96 || s.pans[1].x != -96 {
		t.Fatalf("positional calls = %+v", s.pans)
	}
	if len(s.ids) != 1 || s.ids[0] != 24 {
		t.Fatalf("centred calls = %v", s.ids)
	}
	c.Player.SetFixed(4000, 40000)
	c.Bomb.Start()
	if p := s.pans[len(s.pans)-1]; p.id != 0x26 || p.x != 40 {
		t.Fatalf("bomb sound = %+v, want id 26 at x 40", p)
	}
}

func TestItemPositionalSounds(t *testing.T) {
	c := NewContext(game.NewState(game.CharReimu, game.ShotA, game.DiffNormal))
	s := &panSoundRecord{}
	c.Audio = s
	c.State.Power = 95 // one small P gains a level.
	c.Player.SetFixed(4000, 40000)
	c.SpawnItem(44, 400, item.KindPower, 0, 0)
	c.UpdateItems(false)
	if len(s.pans) != 2 || s.pans[0].id != 0x1d || s.pans[1].id != 0x14 {
		t.Fatalf("item calls = %+v, want level and collect", s.pans)
	}
	for _, p := range s.pans {
		if p.x != 44 {
			t.Errorf("sound %d uses x %g, want the item's x 44, not player x 40", p.id, p.x)
		}
	}
}
