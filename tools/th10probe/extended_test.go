package main

import (
	"encoding/binary"
	"math"
	"testing"

	"th10/trace"
)

func (m *fakeMemory) region(addr uint32) []byte {
	for _, r := range m.regions {
		if r.base == addr {
			return r.data
		}
	}
	panic("fixture region missing")
}

func extendedMemory() (*reader, *fakeMemory) {
	r, m, p := syntheticReader()
	r.ext = true
	g, ptrs := m.region(addrScoreBlock), m.region(addrPointers)
	put32(g, 0, 1000000)
	put32(g, 4, 999999999)
	put32(g, 0xc, 99999)
	put32(g, 0x10, 18)
	put32(g, 0x18, 130)
	put32(g, 0x54, 2)
	put32(g, 0x58, uint32(0xfffffe00))
	put32(p, 0x3cc, uint32(0xfffff8c1)) // -1855
	put32(p, 0x3d0, 42000)
	put32(p, 0x478, 12)
	putFloat(p, 0x47c, 12.25)
	put32(p, 0x4310, 6)
	putFloat(p, 0x4314, 6.5)
	put32(p, 0x4474, 1)
	put32(p, 0x49c+0x40, 1)
	put32(p, 0x49c+0x5c+0x40, 2)
	put32(ptrs, addrItemMgr-addrPointers, 0x03000000)
	items := m.add(0x03000000+itemArrayOff, trace.MaxItemRows*itemStride+4)
	for _, slot := range []int{149, 2197} {
		off := slot * itemStride
		put32(items, off+itemState, 1)
		put32(items, off+itemKind, 8)
		putFloat(items, off+itemPos, -10)
		putFloat(items, off+itemPos+4, 20)
		put32(items, off+itemClock, 1)
		putFloat(items, off+itemClock+4, 1.25)
	}
	put32(items, 2197*itemStride+itemState, 5)
	put32(items, itemLive-itemArrayOff, 1)
	put32(ptrs, addrLaserMgr-addrPointers, 0x04000000)
	mgr := m.add(0x04000000, 0x45c)
	put32(mgr, laserHead, 0x04100000)
	put32(mgr, laserCount, 2)
	for i, addr := range []uint32{0x04100000, 0x04200000} {
		b := m.add(addr, 0x468)
		typeOff := 0x448
		if i == 0 {
			put32(b, 0, laserLineVT)
			put32(b, 8, 0x04200000)
			put32(b, 0xc, 1)
		} else {
			put32(b, 0, laserInfVT)
			typeOff = 0x464
			put32(b, 0xc, 3)
		}
		put32(b, 0x54, uint32(i+8))
		put32(b, 0x14, 3)
		putFloat(b, 0x18, 3.25)
		putFloat(b, 0x24, 10)
		putFloat(b, 0x28, 20)
		putFloat(b, 0x3c, .5)
		putFloat(b, 0x40, 448)
		putFloat(b, 0x44, 2)
		binary.LittleEndian.PutUint16(b[typeOff:], 31)
		binary.LittleEndian.PutUint16(b[typeOff+2:], 65534)
	}
	put32(ptrs, addrSpellPtr-addrPointers, 0x05000000)
	s := m.add(0x05000000, 0x37b0)
	put32(s, 0x3738, 300)
	putFloat(s, 0x373c, 300.25)
	put32(s, 0x3788, 93)
	put32(s, 0x378c, 11)
	put32(s, 0x3790, 1234560)
	put32(s, 0x3794, 1300000)
	put32(s, 0x3798, 1800)
	put32(ptrs, addrStageObj-addrPointers, 0x06000000)
	gui := m.add(0x06000000, 0x9ed0)
	put32(gui, 0x9e78, 12345)
	put32(ptrs, addrBombPtr-addrPointers, 0x07000000)
	bomb := m.add(0x07000000, 0x48)
	put32(bomb, 0x18, 17)
	putFloat(bomb, 0x1c, 17.25)
	put32(bomb, 0x28, 1)
	return r, m
}

func TestDecodeAllExtendedNativeFields(t *testing.T) {
	r, _ := extendedMemory()
	c, err := r.readControl()
	if err != nil {
		t.Fatal(err)
	}
	s, err := r.sampleFrame(c)
	if err != nil {
		t.Fatal(err)
	}
	f, x := s.Frame, s.Frame.Ext
	if f.PScore != 9999999990 || f.PFaith != 999990 || f.PPower != 500 || f.PLife != 2 || f.PInvinc != 6 || f.PBombTimer != 17 || f.PBullets != 1 {
		t.Fatalf("native counters: %+v", f)
	}
	if x.HiScore != 10000000 || x.HiScoreContinues != 2 || x.ShownScore != 123450 || x.FaithHold != 130 || x.FaithRate != 18 || x.Rank != -512 ||
		x.PlayerIX != -1855 || x.PlayerIY != 42000 || x.StateClock != 12 || x.StateClockF != 12.25 || x.InvincF != 6.5 || x.Focus != 1 ||
		!x.BombActive || x.BombClock != 17 || x.BombClockF != 17.25 {
		t.Fatalf("native extension: %+v", x)
	}
	if f.Items != 2 || x.ItemCalc != 1 || len(f.I) != 2 || f.I[1].Slot != 2197 || f.I[1].State != 5 || f.I[0].Type != 8 || f.I[0].X != -10 || f.I[0].Y != 20 || f.I[0].ClockF != 1.25 {
		t.Fatalf("item decoding: calc=%d rows=%+v", x.ItemCalc, f.I)
	}
	if x.Lasers != 2 || len(f.L) != 2 || f.L[0].Kind != 0 || f.L[1].Kind != 1 || f.L[0].State != 1 || f.L[1].State != 3 ||
		f.L[0].ID != 8 || f.L[1].ID != 9 || f.L[0].Color != -2 || f.L[0].Type != 31 || f.L[0].Length != 448 || f.L[0].Width != 2 || f.L[0].Angle != .5 || f.L[0].ClockF != 3.25 {
		t.Fatalf("laser decoding: %+v", f.L)
	}
	card := x.Spell
	if card.ID != 93 || card.Clock != 300 || card.ClockF != 300.25 || card.Bonus != 1234560 || card.Initial != 1300000 || card.Duration != 1800 || !card.Active || !card.Capture || !card.Survival {
		t.Fatalf("spell decoding: %+v", card)
	}
}

func TestExtendedLaserListFailuresAreNotNormalData(t *testing.T) {
	for _, kind := range []string{"vtable", "cycle", "count"} {
		r, m := extendedMemory()
		switch kind {
		case "vtable":
			put32(m.region(0x04100000), 0, 0)
		case "cycle":
			put32(m.region(0x04200000), 8, 0x04100000)
		case "count":
			put32(m.region(0x04000000), laserCount, 3)
		}
		c, _ := r.readControl()
		if _, err := r.sampleFrame(c); err == nil {
			t.Fatalf("accepted malformed laser %s", kind)
		}
	}
}

func TestPooledReadBuffersDoNotChangePreviousSnapshots(t *testing.T) {
	r, m := extendedMemory()
	c, _ := r.readControl()
	a, err := r.sampleFrame(c)
	if err != nil {
		t.Fatal(err)
	}
	putFloat(m.region(0x02000000), addrPlayerX, 30)
	putFloat(m.region(0x03000000+itemArrayOff), 149*itemStride+itemPos, 40)
	b, err := r.sampleFrame(c)
	if err != nil {
		t.Fatal(err)
	}
	if a.Frame.PX == b.Frame.PX || a.Frame.I[0].X == b.Frame.I[0].X || a.Frame.I[0].X != -10 {
		t.Fatal("reused memory changed a prior snapshot")
	}
}

func TestSnapshotChecksBulletsBeyondPrintedSample(t *testing.T) {
	r, m, _ := syntheticReader()
	put32(m.region(addrPointers), addrBulletMgr-addrPointers, 0x08000000)
	pool := m.add(0x08000000+bulletArrayOff, bulletSlots*bulletStride)
	for slot := 0; slot <= trace.MaxBulletRows; slot++ {
		binary.LittleEndian.PutUint16(pool[slot*bulletStride+bulletState:], 1)
	}
	n := 0
	m.before = func(addr uint32, dst []byte) error {
		if addr == 0x08000000+bulletArrayOff {
			n++
			putFloat(pool, trace.MaxBulletRows*bulletStride+bulletPosX, float32(n))
		}
		return nil
	}
	c, _ := r.readControl()
	res := r.sampleSettled(t.Context(), c, testSampleOptions(), (&fakeClock{}).clock())
	if res.Frame.Half == nil || res.Attempts != 4 || len(res.Frame.B) != trace.MaxBulletRows || res.Frame.Bullets != trace.MaxBulletRows+1 {
		t.Fatalf("unprinted motion was not checked: %+v", res)
	}
}

func BenchmarkDecodeExtendedSyntheticMemory(b *testing.B) {
	r, _ := extendedMemory()
	c, _ := r.readControl()
	b.ReportAllocs()
	b.SetBytes(int64(playerSize + 0x64 + 4 + 4 + 0x18 + (0x379c - 0x3734) +
		(trace.MaxItemRows*itemStride + 4) + (laserCount + 4 - laserHead) + 2*0x468))
	for b.Loop() {
		if _, err := r.sampleFrame(c); err != nil {
			b.Fatal(err)
		}
	}
}

func TestNativeFloatBits(t *testing.T) {
	var raw [4]byte
	putFloat(raw[:], 0, math.Float32frombits(0x3eaaaaab))
	if f32(raw[:]) != float64(math.Float32frombits(0x3eaaaaab)) {
		t.Fatal("native float32 bits were recomputed")
	}
}
