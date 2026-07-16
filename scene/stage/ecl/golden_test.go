package ecl

// Golden behavior snapshots: run each stage VM headless and fold enemy
// positions plus full bullet state into per-checkpoint FNV hashes. These lock
// CURRENT behavior so semantic refinements show up as reviewable diffs, not
// silent drift. Regenerate intentionally after a deliberate semantics change:
//
//	go test ./scene/stage/ecl -run TestGoldenStages -update-golden
//
// A checkpoint line also carries enemy/bullet counts so a diff localizes to
// "which 600-frame window" and "did volume or only trajectory change".

import (
	"flag"
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"th10/entity/bullet"
	"th10/game"
)

var updateGolden = flag.Bool("update-golden", false, "rewrite golden stage snapshots")

const (
	goldenFrames     = 3600 // one minute of gameplay
	goldenCheckpoint = 600
)

func stageDigest(eclPath string) []string {
	ctx := newTestContext()
	vm := NewStageVM(eclPath)
	h := fnv.New64a()
	var lines []string
	var buf []byte
	for f := 1; f <= goldenFrames; f++ {
		vm.Update(ctx)
		ctx.Bullets.Update(game.FieldLeft, game.FieldTop, game.FieldRight, game.FieldBottom)
		for _, e := range ctx.Enemies {
			if e == nil || !e.Active {
				continue
			}
			buf = fmt.Appendf(buf[:0], "e%.2f,%.2f;", e.X, e.Y)
			h.Write(buf)
		}
		ctx.Bullets.Each(func(b *bullet.Bullet) {
			buf = fmt.Appendf(buf[:0], "b%.2f,%.2f,%.3f,%.2f,%d,%d;", b.X, b.Y, b.Angle, b.Speed, int(b.Type), b.Color)
			h.Write(buf)
		})
		if f%goldenCheckpoint == 0 {
			lines = append(lines, fmt.Sprintf("frame %04d hash %016x enemies %d bullets %d",
				f, h.Sum64(), activeEnemies(ctx), ctx.Bullets.ActiveCount()))
		}
	}
	return lines
}

func TestGoldenStages(t *testing.T) {
	for _, st := range []string{"stage01", "stage02", "stage03", "stage04", "stage05", "stage06"} {
		t.Run(st, func(t *testing.T) {
			got := stageDigest("ecl/" + st + ".ecl")
			gold := filepath.Join("testdata", "golden_"+st+".txt")
			if *updateGolden {
				if err := os.MkdirAll("testdata", 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(gold, []byte(strings.Join(got, "\n")+"\n"), 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}
			raw, err := os.ReadFile(gold)
			if err != nil {
				t.Skipf("no golden file yet (%v); run with -update-golden", err)
			}
			want := strings.Split(strings.TrimSpace(string(raw)), "\n")
			for i, g := range got {
				if i >= len(want) {
					t.Fatalf("golden has %d checkpoints, produced %d", len(want), len(got))
				}
				if g != want[i] {
					t.Fatalf("behavior diverged at checkpoint %d:\n  got  %s\n  want %s", i, g, want[i])
				}
			}
		})
	}
}
