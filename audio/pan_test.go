package audio

import (
	"math"
	"testing"
)

func TestSEPan(t *testing.T) {
	for _, tc := range []struct {
		x    float32
		want int32
	}{
		{0, 0}, {192, 1000}, {-192, -1000}, {96, 500}, {-96, -500},
		{1, 5}, {-1, -5}, {0.1, 0}, {-0.1, 0}, {0.2, 1}, {-0.2, -1},
		{191.99998, 999}, {-191.99998, -999},
	} {
		if got := SEPan(tc.x); got != tc.want {
			t.Errorf("SEPan(%g) = %d, want %d", tc.x, got, tc.want)
		}
	}
}

func TestSEPanGains(t *testing.T) {
	for _, tc := range []struct {
		pan         int32
		left, right float64
	}{{0, 1, 1}, {1000, 0.31622776601683794, 1}, {-1000, 1, 0.31622776601683794},
		{2000, 0.1, 1}, {-2000, 1, 0.1}} {
		l, r := sePanGains(tc.pan)
		if math.Abs(l-tc.left) > 1e-15 || math.Abs(r-tc.right) > 1e-15 {
			t.Errorf("pan %d gains = %.17g/%.17g, want %.17g/%.17g", tc.pan, l, r, tc.left, tc.right)
		}
	}
}

func TestSEQueuePanAndAux(t *testing.T) {
	m := NewManager(nil)
	if m.seAux[2] != -1 {
		t.Fatal("aux attribute was not initialized to -1")
	}
	m.PlaySEIDAt(2, 192)
	m.PlaySEIDAt(2, -96)
	m.PlaySEID(2)
	if len(m.seQueue) != 1 || m.seQueue[0].count != 3 || m.seQueue[0].pan() != 166 {
		t.Fatalf("queue = %+v, want one id with three requests and pan trunc(500/3)=166", m.seQueue)
	}
	if m.seAux[2] != 5 {
		t.Fatalf("aux = %d, want table +6 value 5", m.seAux[2])
	}
	m.Update()
	if len(m.seQueue) != 0 || m.seAux[2] != 5 {
		t.Fatal("frame flush did not clear requests or incorrectly cleared the unused attribute")
	}
	m.PlaySEIDAt(2, -0.2)
	m.PlaySEID(2)
	if m.seQueue[0].pan() != 0 {
		t.Fatalf("negative average = %d, want trunc(-1/2)=0", m.seQueue[0].pan())
	}
}

func TestSEQueueLimits(t *testing.T) {
	m := NewManager(nil)
	m.PlaySEID(-1)
	m.PlaySEID(len(seTable))
	for id := 0; id <= seQueueMax; id++ {
		m.PlaySEID(id)
	}
	if len(m.seQueue) != seQueueMax || m.seAux[seQueueMax] != -1 {
		t.Fatalf("queue limit = %d, aux for dropped id = %d", len(m.seQueue), m.seAux[seQueueMax])
	}
	for range sePanMax - 1 {
		m.PlaySEIDAt(0, 192)
	}
	m.PlaySEIDAt(0, -192) // request 129 must not change the first 128 pans.
	q := m.seQueue[0]
	if q.count != sePanMax || q.sum != 127000 || q.pan() != 992 {
		t.Fatalf("128-pan limit = %+v, average %d", q, q.pan())
	}
}
