package sim

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"th10/entity/playerbullet"
	"th10/replay"
	"th10/scene/stage"
)

func marisaReplay(t *testing.T, name string) (*replay.Replay, Config) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", ".tmp", "[th10]", "replay", name))
	if err != nil {
		t.Skipf("original replay unavailable: %v", err)
	}
	rp, err := replay.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := ConfigFor(rp, 0)
	if err != nil {
		t.Fatal(err)
	}
	return rp, cfg
}

func TestMarisaBLasersStartOnFirstFrame(t *testing.T) {
	rp, cfg := marisaReplay(t, "th10_03.rpy")
	r := New(cfg, recordFactory)
	r.OnFrame = func(st *stage.Stage) {
		c := stage.ExportContext(st)
		beams := 0
		c.Player.Bullets.Each(func(b *playerbullet.Bullet) {
			if b.ScriptID == 10 {
				beams++
				if b.Age != 1 || !c.World.Contains(b.VM) || !c.World.Contains(b.ExtraVM) {
					t.Fatal("first-phase beam lost its clock or either World handle")
				}
			}
		})
		// FUN_00417870 creates DAT_00477704 before calc; FUN_00428280
		// must not interpret a not-yet-bound Go hook as a missing manager.
		if beams != 4 {
			t.Fatalf("first-frame beams=%d, want 4", beams)
		}
	}
	if err := r.Run(rp.Chapters[0], io.Discard, 1); err != nil {
		t.Fatal(err)
	}
}

func TestMarisaBFrame252LaserDamage(t *testing.T) {
	rp, cfg := marisaReplay(t, "th10_03.rpy")
	r := New(cfg, recordFactory)
	r.OnFrame = func(st *stage.Stage) {
		c := stage.ExportContext(st)
		if c.State.Frame != 252 {
			return
		}
		enemies := c.ManagerOrder()
		// orig_th10_03 frame 252: the source-1 beam deals 9 damage on
		// FUN_00427c70's fourth-tick gate, leaving enemy[2] at 11 HP.
		if len(enemies) != 13 {
			t.Fatalf("frame 252 enemies=%d, want 13", len(enemies))
		}
		if enemies[2].HP != 11 {
			t.Fatalf("frame 252 enemy[2] HP=%d, want 11", enemies[2].HP)
		}
	}
	if err := r.Run(rp.Chapters[0], io.Discard, 252); err != nil {
		t.Fatal(err)
	}
}

func TestMarisaARespawnTrailAndRocketHit(t *testing.T) {
	rp, cfg := marisaReplay(t, "th10_04.rpy")
	r := New(cfg, recordFactory)
	want := map[int]int{3339: 767, 3340: 767, 3341: 766, 3342: 766, 3343: 765}
	r.OnFrame = func(st *stage.Stage) {
		c := stage.ExportContext(st)
		hp, check := want[c.State.Frame]
		if !check {
			return
		}
		enemies := c.ManagerOrder()
		if len(enemies) != 5 {
			t.Fatalf("frame %d enemies=%d, want 5", c.State.Frame, len(enemies))
		}
		// orig_th10_04: the first post-respawn rocket hits at 3341,
		// then its newly created area skips countdown 12 at 3342.
		if enemies[2].HP != hp {
			t.Errorf("frame %d enemy[2] HP=%d, want %d", c.State.Frame, enemies[2].HP, hp)
		}
	}
	if err := r.Run(rp.Chapters[0], io.Discard, 3343); err != nil {
		t.Fatal(err)
	}
}
