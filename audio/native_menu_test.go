package audio

import "testing"

func TestNativeVolumeCurves(t *testing.T) {
	for _, test := range []struct {
		percent uint8
		base    int32
		se      bool
		want    int32
	}{
		{0, 0, false, -10000}, {100, 0, false, 0}, {50, 0, false, -1250},
		{0, -500, true, -10000}, {100, -500, true, -500}, {50, -500, true, -1063},
	} {
		if got := NativeVolumeDB(test.percent, test.base, test.se); got != test.want {
			t.Fatalf("%+v got %d", test, got)
		}
	}
	m := NewManager(nil)
	m.SetNativeVolumes(100, 80)
	if !m.nativeVolume || m.nativeBGM != 100 || m.nativeSE != 80 || m.seVolume != 1 || m.bgmVolume != 1 {
		t.Fatal("native percentage setup differs")
	}
	m.PauseBGM()
	if !m.bgmPaused {
		t.Fatal("pause flag missing")
	}
	m.ResumeBGM()
	if m.bgmPaused {
		t.Fatal("unpause flag held")
	}
}
