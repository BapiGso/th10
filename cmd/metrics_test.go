package main

import (
	"testing"
	"time"
)

func TestMetricsSeparate60TPSFromHighRefreshDraw(t *testing.T) {
	for _, hz := range []int{60, 120, 144} {
		var stats frameStats
		start := time.Unix(0, 0)
		stats.reset(start)
		previous := 0
		for refresh := 1; refresh <= hz; refresh++ {
			ticks := refresh * 60 / hz
			for ; previous < ticks; previous++ {
				stats.update(2 * time.Millisecond)
			}
			stats.draw(time.Millisecond)
		}
		if !stats.publish(start.Add(time.Second)) {
			t.Fatal("one-second sample was not published")
		}
		m := stats.view()
		if m.TPS != 60 || m.FPS != float64(hz) || m.UpdateMeanMS != 2 || m.DrawMeanMS != 1 || m.UpdateMaxMS != 2 || m.DrawMaxMS != 1 {
			t.Fatalf("%d Hz sample: %+v", hz, m)
		}
	}
}

func TestMetricsLastAndWindowMaximum(t *testing.T) {
	var stats frameStats
	start := time.Unix(0, 0)
	stats.reset(start)
	stats.update(8 * time.Millisecond)
	stats.update(time.Millisecond)
	stats.draw(3 * time.Millisecond)
	stats.setHeap(1234)
	if stats.publish(start.Add(time.Second / 2)) {
		t.Fatal("published a partial window")
	}
	stats.publish(start.Add(time.Second))
	if m := stats.view(); m.UpdateMS != 1 || m.UpdateMaxMS != 8 || m.UpdateMeanMS != 4.5 || m.DrawMS != 3 || m.HeapBytes != 1234 {
		t.Fatalf("last/mean/max differs: %+v", m)
	}
}
