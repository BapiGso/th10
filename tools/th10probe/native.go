package main

import (
	"encoding/binary"
	"fmt"
	"math"

	"th10/trace"
)

// Addresses are for the hash-checked v1.00a image. See probe_ext.md for
// the individual writers; none of these addresses is written by this tool.
const (
	imageBase    = 0x400000
	ExpectedHash = "2f14760b6fbbf57549541583283badb9a19a4222b90f0a146d5aa17f01dc9040"

	addrScoreBlock   = 0x00474c40
	addrFrameCounter = 0x00474c88 // FUN_00418190, calc priority 0x0a
	addrPointers     = 0x004776e0
	addrBulletMgr    = 0x004776f0
	addrEnemyMgr     = 0x00477704
	addrStageObj     = 0x0047770c // GUI, not DAT_00477810
	addrStagePtr     = 0x00477810
	addrPlayerPtr    = 0x00477834
	addrItemMgr      = 0x00477818
	addrLaserMgr     = 0x0047781c
	addrSpellPtr     = 0x004776f4
	addrBombPtr      = 0x004776ec
	addrReplayPtr    = 0x00477838
	addrReplayName   = 0x00477710 // char[256], FUN_0042cdf0 / replay menu
	addrScene        = 0x00491fb4

	addrDrawSkip     = 0x00491d66
	addrDrawCount    = 0x00492504 // FUN_00439390, after the whole calc chain
	addrLoopStart    = 0x00492528 // main-loop object +0x38
	addrLoopTime     = 0x004923a0 // FUN_00439390, after draw/Present
	addrPlayerX      = 0x3c0
	addrPlayerY      = 0x3c4
	addrPlayerState  = 0x458
	addrPlayerInvinc = 0x4310
	playerSize       = 0x4478

	enemyListHead = 0x58
	enemyMaxNodes = 4096
	enemyPosX     = 0x1068
	enemyPosY     = 0x106c
	enemyHP       = 0x23fc
	enemyFlags    = 0x2480

	stageMSG  = 0x9eb8
	msgSize   = 0x90
	msgClock  = 0x1c
	msgWait   = 0x30
	msgPC     = 0x5c
	msgSignal = 0x78

	bulletArrayOff = 0x60
	bulletStride   = 0x7f0
	bulletSlots    = 2000
	bulletPosX     = 0x3b4
	bulletPosY     = 0x3b8
	bulletSpeed    = 0x3d8
	bulletAngle    = 0x3e4
	bulletType     = 0x7ea
	bulletSprite   = 0x7ec
	bulletState    = 0x446
)

// memoryReader has deliberately no write, suspend, input or window API.
// Unit tests implement it with synthetic byte-addressed snapshots.
type memoryReader interface {
	ReadMemory(addr uint32, dst []byte) error
}

type reader struct {
	mem     memoryReader
	ext     bool
	player  []byte
	bullets []byte
	items   []byte
}

func (r *reader) block(addr uint32, n int, storage *[]byte) ([]byte, error) {
	if len(*storage) != n {
		*storage = make([]byte, n)
	}
	if err := r.mem.ReadMemory(addr, *storage); err != nil {
		return nil, fmt.Errorf("read 0x%08x (%d bytes): %w", addr, n, err)
	}
	return *storage, nil
}

func (r *reader) bytes(addr uint32, n int) ([]byte, error) {
	b := make([]byte, n)
	if err := r.mem.ReadMemory(addr, b); err != nil {
		return nil, fmt.Errorf("read 0x%08x (%d bytes): %w", addr, n, err)
	}
	return b, nil
}

func (r *reader) u32(addr uint32) (uint32, error) {
	b, err := r.bytes(addr, 4)
	if err != nil {
		return 0, err
	}
	return le32(b), nil
}

func (r *reader) i32(addr uint32) (int32, error) {
	v, err := r.u32(addr)
	return int32(v), err
}

func le32(b []byte) uint32 { return binary.LittleEndian.Uint32(b) }
func le16(b []byte) uint16 { return binary.LittleEndian.Uint16(b) }
func i32(b []byte) int     { return int(int32(le32(b))) }
func f32(b []byte) float64 { return float64(math.Float32frombits(le32(b))) }

type control struct {
	Frame                               int32
	Stage, Player, Bullet, Enemy, GUI   uint32
	Item, Laser, Spell, Bomb            uint32
	Replay, Header                      uint32
	Scene, Request, ReplayMode, Chapter int32
	StageFlags, RunFlags                uint32
	LoopStart, LoopTime                 uint64
	DrawCount                           int8
	DrawSkip                            uint8
}

func (a control) sameFrame(b control) bool {
	return a.Frame == b.Frame && a.Stage == b.Stage && a.Player == b.Player &&
		a.Bullet == b.Bullet && a.Enemy == b.Enemy && a.GUI == b.GUI &&
		a.Item == b.Item && a.Laser == b.Laser && a.Spell == b.Spell && a.Bomb == b.Bomb &&
		a.Replay == b.Replay && a.Header == b.Header && a.Chapter == b.Chapter && a.Scene == b.Scene
}

func (a control) sameStage(b control) bool {
	return a.Stage == b.Stage && a.Replay == b.Replay && a.Header == b.Header && a.Chapter == b.Chapter
}

func (c control) playable() bool {
	return c.Stage != 0 && c.Scene == 7 && c.StageFlags&4 == 0 && c.Frame >= 1
}

// readControl brackets each snapshot. The draw counter is a completion
// pulse only with no draw skipping; loop timestamps are later witnesses.
func (r *reader) readControl() (c control, err error) {
	if c.Frame, err = r.i32(addrFrameCounter); err != nil {
		return c, err
	}
	p, err := r.bytes(addrPointers, 0x178)
	if err != nil {
		return c, err
	}
	c.Stage = le32(p[addrStagePtr-addrPointers:])
	c.Player = le32(p[addrPlayerPtr-addrPointers:])
	c.Bullet = le32(p[addrBulletMgr-addrPointers:])
	c.Enemy = le32(p[addrEnemyMgr-addrPointers:])
	c.GUI = le32(p[addrStageObj-addrPointers:])
	c.Item = le32(p[addrItemMgr-addrPointers:])
	c.Laser = le32(p[addrLaserMgr-addrPointers:])
	c.Spell = le32(p[addrSpellPtr-addrPointers:])
	c.Bomb = le32(p[addrBombPtr-addrPointers:])
	c.Replay = le32(p[addrReplayPtr-addrPointers:])
	scene, err := r.bytes(addrScene, 8)
	if err != nil {
		return c, err
	}
	c.Scene, c.Request = int32(le32(scene)), int32(le32(scene[4:]))
	if c.Stage != 0 {
		flags, err := r.u32(c.Stage + 0x58)
		if err != nil {
			return c, err
		}
		c.StageFlags = flags
	}
	if c.RunFlags, err = r.u32(addrScoreBlock + 0x60); err != nil {
		return c, err
	}
	if c.Replay != 0 {
		mode, err := r.i32(c.Replay + 0x10)
		if err != nil {
			return c, err
		}
		c.ReplayMode = mode
		if c.Chapter, err = r.i32(c.Replay + 0x1d0); err != nil {
			return c, err
		}
		if c.Chapter >= 1 && c.Chapter <= 7 {
			if c.Header, err = r.u32(c.Replay + 0xb0 + uint32(c.Chapter)*0x24); err != nil {
				return c, err
			}
		}
	}
	loop, err := r.bytes(addrDrawCount, addrLoopStart+8-addrDrawCount)
	if err != nil {
		return c, err
	}
	c.DrawCount = int8(loop[0])
	c.LoopStart = binary.LittleEndian.Uint64(loop[addrLoopStart-addrDrawCount:])
	timing, err := r.bytes(addrLoopTime, 8)
	if err != nil {
		return c, err
	}
	c.LoopTime = binary.LittleEndian.Uint64(timing)
	skip, err := r.bytes(addrDrawSkip, 1)
	if err != nil {
		return c, err
	}
	c.DrawSkip = skip[0]
	return c, nil
}

func calcWitness(first, now control) bool {
	// FUN_00439390 0x439478, 0x43951b, and the next 0x43939f.
	return now.DrawSkip == 0 && now.DrawCount > 0 ||
		now.LoopStart != first.LoopStart || now.LoopTime != first.LoopTime
}

type bulletSample struct {
	Slot, State, Type, Colour int
	X, Y, Angle, Speed        uint32
}

type snapshot struct {
	Frame   *trace.Frame
	Bullets []bulletSample // includes slots beyond the printed 256-row sample
}

func (r *reader) sampleFrame(c control) (*snapshot, error) {
	f := &trace.Frame{Index: int(c.Frame)}
	s := &snapshot{Frame: f}
	resources, err := r.bytes(addrScoreBlock, 0x64)
	if err != nil {
		return s, err
	}
	f.PScore = int64(int32(le32(resources[4:]))) * 10
	f.PPower = int(int16(le16(resources[8:]))) * 5 // FUN_00418930 writes only a word
	f.PFaith = int64(int32(le32(resources[0xc:]))) * 10
	f.PLife = i32(resources[0x30:])
	// FUN_00406240/FUN_0041d3d0 do not maintain a graze counter.
	var player []byte
	if c.Player != 0 {
		p, err := r.block(c.Player, playerSize, &r.player)
		if err != nil {
			return s, err
		}
		f.PX, f.PY = f32(p[addrPlayerX:]), f32(p[addrPlayerY:])
		f.PState, f.PInvinc = i32(p[addrPlayerState:]), i32(p[addrPlayerInvinc:])
		player = p
	}
	if c.Enemy != 0 {
		node, err := r.u32(c.Enemy + enemyListHead)
		if err != nil {
			return s, err
		}
		seen := make(map[uint32]bool)
		for node != 0 {
			if seen[node] || len(seen) >= enemyMaxNodes {
				return s, fmt.Errorf("unstable enemy list at 0x%08x", node)
			}
			seen[node] = true
			pair, err := r.bytes(node, 8)
			if err != nil {
				return s, err
			}
			e := le32(pair)
			node = le32(pair[4:])
			if e == 0 {
				continue
			}
			f.Enemies++
			if len(f.E) >= trace.MaxEnemyRows {
				continue
			}
			b, err := r.bytes(e+enemyPosX, enemyFlags+4-enemyPosX)
			if err != nil {
				return s, err
			}
			f.E = append(f.E, trace.EnemyRow{X: f32(b), Y: f32(b[4:]),
				HP: i32(b[enemyHP-enemyPosX:]), Active: le32(b[enemyFlags-enemyPosX:])&0x20000 == 0})
		}
	}
	if c.Bullet != 0 {
		pool, err := r.block(c.Bullet+bulletArrayOff, bulletSlots*bulletStride, &r.bullets)
		if err != nil {
			return s, err
		}
		for i := 0; i < bulletSlots; i++ {
			b := pool[i*bulletStride:]
			state := int(le16(b[bulletState:]))
			if state == 0 {
				continue
			}
			f.Bullets++
			s.Bullets = append(s.Bullets, bulletSample{Slot: i, State: state,
				Type: int(int16(le16(b[bulletType:]))), Colour: int(int16(le16(b[bulletSprite:]))),
				X: le32(b[bulletPosX:]), Y: le32(b[bulletPosY:]),
				Angle: le32(b[bulletAngle:]), Speed: le32(b[bulletSpeed:])})
			if len(f.B) < trace.MaxBulletRows {
				f.B = append(f.B, trace.BulletRow{X: f32(b[bulletPosX:]), Y: f32(b[bulletPosY:]),
					Angle: f32(b[bulletAngle:]), Speed: f32(b[bulletSpeed:]),
					Type: int(int16(le16(b[bulletType:]))), Color: int(int16(le16(b[bulletSprite:])))})
			}
		}
	}
	if c.GUI != 0 {
		mp, err := r.u32(c.GUI + stageMSG)
		if err != nil {
			return s, err
		}
		if mp != 0 {
			b, err := r.bytes(mp, msgSize)
			if err != nil {
				return s, err
			}
			in, err := r.bytes(le32(b[msgPC:]), 3)
			if err != nil {
				return s, err
			}
			f.MSG = &trace.MSGRow{ID: i32(b), Clock: i32(b[msgClock:]), Wait: i32(b[msgWait:]),
				Signal: i32(b[msgSignal:]), At: int(le16(in)), Op: int(in[2])}
		}
	}
	if r.ext {
		if err := r.sampleExtended(c, f, resources, player); err != nil {
			return s, err
		}
	}
	return s, nil
}
