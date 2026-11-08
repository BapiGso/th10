package audio

import "math"

// seWav is the wav name table at 0x474b40.
var seWav = [...]string{
	"se_plst00.wav", "se_enep00.wav", "se_pldead00.wav", "se_power0.wav",
	"se_power1.wav", "se_tan00.wav", "se_tan01.wav", "se_tan02.wav",
	"se_ok00.wav", "se_cancel00.wav", "se_select00.wav", "se_gun00.wav",
	"se_cat00.wav", "se_lazer00.wav", "se_lazer01.wav", "se_enep01.wav",
	"se_damage00.wav", "se_item00.wav", "se_kira00.wav", "se_kira01.wav",
	"se_kira02.wav", "se_timeout.wav", "se_graze.wav", "se_powerup.wav",
	"se_pause.wav", "se_cardget.wav", "se_option.wav", "se_damage01.wav",
	"se_timeout2.wav", "se_invalid.wav", "se_slash.wav", "se_ch00.wav",
	"se_ch01.wav", "se_hint00.wav", "se_extend.wav", "se_bonus3.wav",
	"se_water.wav",
}

// seEntry is one sound of the table at 0x4749c8 (8 bytes): its wav and its
// volume in hundredths of a decibel. aux (+6) is copied to manager+0x408
// by FUN_0043dc90/0043dd10; this executable has no runtime reader of it.
type seEntry struct {
	wav int8
	vol int16
	aux int16
}

// seTable is the table at 0x4749c8, indexed by the sound id the game and the
// scripts pass (ECL 336, the bullet template sounds, ...).
var seTable = [...]seEntry{
	{0, -1900, 0}, {0, -2100, 0}, {1, -1200, 5}, {1, -1500, 5},
	{2, -1100, 100}, {3, -700, 100}, {4, -700, 100}, {5, -1900, 50},
	{6, -2200, 50}, {7, -2400, 50}, {8, -500, 100}, {9, -400, 100},
	{10, -800, 10}, {11, -1500, 10}, {12, -1000, 100}, {5, -1100, 50},
	{13, -1300, 50}, {14, -1400, 50}, {15, -900, 100}, {16, -880, 0},
	{17, -1500, 0}, {5, -300, 20}, {6, -1800, 20}, {7, -1800, 20},
	{18, -1100, 50}, {19, -1300, 50}, {20, -1500, 50}, {21, -500, 100},
	{22, -1100, 20}, {23, -800, 90}, {22, -1200, 20}, {18, -500, 50},
	{24, -800, 100}, {25, -800, 100}, {26, -800, 100}, {27, -500, 0},
	{28, -300, 100}, {29, 0, 100}, {30, 0, 100}, {30, -600, 100},
	{8, -300, 100}, {31, -300, 100}, {32, -300, 100}, {33, -300, 100},
	{34, -100, 100}, {35, 0, 100}, {36, -800, 50},
}

// seQueueMax is how many distinct sounds one frame can start.
const seQueueMax = 12

// FUN_0043dd10 (0x43dd50) collects at most 128 pans for one sound per frame.
const sePanMax = 128

type seRequest struct {
	id    int
	count int32
	sum   int32
}

// pan is the signed sum / count in FUN_0043ddf0 (0x43e28d), toward zero.
func (r seRequest) pan() int32 { return r.sum / r.count }

// SEName returns the wav sound id plays, if the id is in the table.
func SEName(id int) (string, bool) {
	if id < 0 || id >= len(seTable) {
		return "", false
	}
	return seWav[seTable[id].wav], true
}

// seGain is the table volume as an amplitude factor, 10^(vol/2000).
func seGain(id int) float64 {
	return math.Pow(10, float64(seTable[id].vol)/2000)
}

// PlaySEID is FUN_0043dc90 with pan 0. Update starts each queued id once,
// after all its requests for this frame have been collected.
func (m *Manager) PlaySEID(id int) {
	m.queueSE(id, 0)
}

// PlaySEIDAt is FUN_0043dd10: x is relative to the playfield's top-centre.
func (m *Manager) PlaySEIDAt(id int, x float32) {
	m.queueSE(id, SEPan(x))
}

// queueSE is FUN_0043dc90/0043dd10: first appearance copies aux, subsequent
// appearances only append pans, and a thirteenth distinct id is dropped.
func (m *Manager) queueSE(id int, pan int32) {
	if _, ok := SEName(id); !ok {
		return
	}
	for i := range m.seQueue {
		q := &m.seQueue[i]
		if q.id == id {
			if q.count < sePanMax {
				q.sum += pan
				q.count++
			}
			return
		}
	}
	if len(m.seQueue) >= seQueueMax {
		return
	}
	m.seQueue = append(m.seQueue, seRequest{id: id, count: 1, sum: pan})
	m.seAux[id] = seTable[id].aux // 0x43dd8b, deliberately not a cooldown/priority.
}

// flushSE is FUN_0043ddf0 (0x43e22b..0x43e345): one voice per table id,
// stopped and rewound on each queued play, with the average pan. Different
// ids have separate voices even when they refer to the same wav.
func (m *Manager) flushSE() {
	for _, q := range m.seQueue {
		if p := m.seByID[q.id]; p != nil {
			p.PauseAndStopReading()
		}
		name, _ := SEName(q.id)
		gain := seGain(q.id)
		if m.nativeVolume {
			gain = nativeGain(m.nativeSE, int32(seTable[q.id].vol), true)
		}
		m.seByID[q.id] = m.playSEPan(name, gain, q.pan())
	}
	m.seQueue = m.seQueue[:0]
}
