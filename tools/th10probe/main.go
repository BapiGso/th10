// Command th10probe samples a running TH10 process read-only and writes a
// per-frame trace in the shared trace format (package trace).
//
// It attaches with PROCESS_VM_READ only — no writes, no injection, no code
// patching — so the original process is never modified. Run the original,
// start a replay (or play), then run this tool while it plays.
//
// Usage:
//
//	th10probe -pid 1234 -frames 3600 -out orig.trace
//	th10probe -exe "C:\path\to\th10.exe" ...   # find the pid by module path
//
// The address map below comes from Ghidra decompilations of th10.exe
// (SHA-256 2f14760b6fbbf57549541583283badb9a19a4222b90f0a146d5aa17f01dc9040,
// un-ASLR'd image base 0x400000). The tool verifies the module base and the
// executable hash before trusting any address.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
	"th10/trace"
)

// Known image base of th10.exe (no relocation applied at runtime for this build).
const imageBase = 0x400000

// ExpectedHash is the SHA-256 of the reference th10.exe.
const ExpectedHash = "2f14760b6fbbf57549541583283badb9a19a4222b90f0a146d5aa17f01dc9040"

// Global addresses (absolute, image-base relative offsets from Ghidra).
const (
	addrPlayerPtr   = 0x00477834 // *Player
	addrBulletMgr   = 0x004776f0 // *bullet manager (FUN_004065c0, FUN_004067d0)
	addrEnemyMgr    = 0x00477704 // *enemy manager (FUN_0040d750)
	addrScore       = 0x00474c40 // score
	addrLife        = 0x00474c70 // lives
	addrPower       = 0x00474c48 // power (raw, 0..100)
	addrFaith       = 0x00474c90 // faith
	addrGraze       = 0x00474c84 // graze counter (candidate)
	addrPlayerX     = 0x3c0      // offset from player base (game space, FUN_004073e0)
	addrPlayerY     = 0x3c4
	addrPlayerState = 0x478 // player state index (candidate)
)

// Frame-counter candidates. FUN_00418190 increments DAT_00474c88 and
// DAT_00474c8c once per main-loop iteration and FUN_00417870 zeroes both at
// stage start, so those are the stage frame clocks. DAT_00491c10 looked like a
// counter but is a millisecond timestamp (passed to ANM helpers).
var frameCounterCandidates = []struct {
	name string
	addr uint32
}{
	{"DAT_00474c88", 0x00474c88},
	{"DAT_00474c8c", 0x00474c8c},
	{"DAT_00491c10", 0x00491c10},
	{"DAT_00474c98", 0x00474c98}, // dynamic rank (+1/60f)
}

// addrFrameCounter is the stage frame clock, confirmed by probeFrameCounters.
const addrFrameCounter = 0x00474c88

// Enemy manager (FUN_0040d750): +0x58 points at the first list node, a node
// is {enemy*, next*}. Enemy fields (docs/research/m2_motion.md section 5):
// EnemyData sits at enemy+0x103c.
const (
	enemyListHead = 0x58
	enemyMaxNodes = 4096   // guards against a torn list read
	enemyPosX     = 0x1068 // final position (A+B), EnemyData+0x2c
	enemyPosY     = 0x106c
	enemyHP       = 0x23fc
	enemyFlags    = 0x2480
)

// Bullet manager (FUN_004065c0/FUN_004067d0): 2000 bullets inline from +0x60,
// stride 0x7f0, then a sentinel slot with state 5.
const (
	bulletArrayOff = 0x60
	bulletStride   = 0x7f0
	bulletSlots    = 2000
	bulletPosX     = 0x3b4
	bulletPosY     = 0x3b8
	bulletSpeed    = 0x3d8
	bulletAngle    = 0x3e4
	bulletType     = 0x7ea // short (candidate)
	bulletSprite   = 0x7ec // short (candidate)
	bulletState    = 0x446 // short: 0 free, 1 normal, 2 spawning, ...
)

func main() {
	os.Exit(run())
}

func run() int {
	var (
		pid      = flag.Uint("pid", 0, "target process id")
		exePath  = flag.String("exe", "", "path to th10.exe (to locate the pid and verify the hash)")
		frames   = flag.Int("frames", 3600, "sample up to this stage frame")
		outPath  = flag.String("out", "-", "output trace path (- = stdout)")
		pollUS   = flag.Int("poll-us", 500, "frame-poll interval in microseconds")
		settleUS = flag.Int("settle-us", 3000, "wait after a frame tick before sampling, so the frame's update has finished")
		noWait   = flag.Bool("no-wait-start", false, "sample from the current frame instead of waiting for a stage start")
		skipHash = flag.Bool("skip-hash", false, "skip the exe SHA-256 check")
		counters = flag.Bool("probe-counters", false, "print which frame-counter candidate ticks, then exit")
	)
	flag.Parse()

	target, err := resolveTarget(uint32(*pid), *exePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "th10probe:", err)
		return 2
	}
	if !*skipHash {
		if err := verifyHash(target.exePath); err != nil {
			fmt.Fprintln(os.Stderr, "th10probe:", err)
			return 2
		}
	}

	h, err := windows.OpenProcess(windows.PROCESS_VM_READ|windows.PROCESS_QUERY_INFORMATION, false, target.pid)
	if err != nil {
		fmt.Fprintln(os.Stderr, "th10probe: OpenProcess:", err)
		return 2
	}
	defer windows.CloseHandle(h)

	base, err := moduleBase(target.pid, target.exePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "th10probe:", err)
		return 2
	}
	if base != imageBase {
		fmt.Fprintf(os.Stderr, "th10probe: module base 0x%x != expected 0x%x; addresses are image-relative and this build may be relocated\n",
			base, imageBase)
		return 2
	}

	out := os.Stdout
	if *outPath != "-" {
		f, err := os.Create(*outPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, "th10probe:", err)
			return 2
		}
		defer f.Close()
		out = f
	}

	rd := &reader{h: h}
	if *counters {
		probeFrameCounters(rd, 3*time.Second)
		return 0
	}
	tw := trace.NewWriter(out)
	poll := time.Duration(*pollUS) * time.Microsecond

	fmt.Fprintf(os.Stderr, "th10probe: attached to pid %d, sampling up to frame %d\n", target.pid, *frames)

	// Frames are indexed by the stage frame clock, so dropped samples leave
	// gaps that th10diff skips instead of shifting every later frame. Unless
	// told otherwise, wait for the clock to restart (a new stage) so the
	// trace begins at the stage's first frame.
	lastFrame := int32(-1)
	if !*noWait {
		fmt.Fprintln(os.Stderr, "th10probe: waiting for a stage start (start the replay now)")
		prev, _ := rd.i32(addrFrameCounter)
		for {
			fc, err := rd.i32(addrFrameCounter)
			if err != nil {
				fmt.Fprintln(os.Stderr, "th10probe: read frame counter:", err)
				return 2
			}
			if fc >= 0 && fc <= 2 && fc != prev { // zeroed or first tick
				break
			}
			prev = fc
			time.Sleep(poll)
		}
	}
	sampled, torn := 0, 0
	settle := time.Duration(*settleUS) * time.Microsecond
	deadline := time.Now().Add(30 * time.Minute)
	for lastFrame < int32(*frames) {
		if time.Now().After(deadline) {
			fmt.Fprintln(os.Stderr, "th10probe: timeout waiting for frames")
			break
		}
		fc, err := rd.i32(addrFrameCounter)
		if err != nil {
			fmt.Fprintln(os.Stderr, "th10probe: read frame counter:", err)
			return 2
		}
		if fc == lastFrame || fc < 0 {
			time.Sleep(poll)
			continue
		}
		if lastFrame >= 0 && fc < lastFrame {
			fmt.Fprintf(os.Stderr, "th10probe: frame clock restarted (%d -> %d); stopping\n", lastFrame, fc)
			break
		}
		if lastFrame >= 0 && fc != lastFrame+1 {
			fmt.Fprintf(os.Stderr, "th10probe: frame jump %d -> %d (dropped %d)\n",
				lastFrame, fc, fc-lastFrame-1)
		}
		lastFrame = fc
		if fc > int32(*frames) {
			break
		}
		time.Sleep(settle)
		f, err := rd.sampleFrame(int(fc))
		if err != nil {
			fmt.Fprintln(os.Stderr, "th10probe: sample:", err)
			return 2
		}
		// A sample that straddles the next frame's update mixes two frames;
		// drop it rather than report a false divergence.
		if after, err := rd.i32(addrFrameCounter); err == nil && after != fc {
			torn++
			continue
		}
		sampled++
		tw.WriteFrame(f)
	}
	if torn > 0 {
		fmt.Fprintf(os.Stderr, "th10probe: dropped %d torn samples (lower -settle-us?)\n", torn)
	}
	if err := tw.Flush(); err != nil {
		fmt.Fprintln(os.Stderr, "th10probe:", err)
		return 2
	}
	fmt.Fprintf(os.Stderr, "th10probe: wrote %d frames\n", sampled)
	return 0
}

// probeFrameCounters prints which candidate global actually ticks, so the
// frame-clock address is chosen from evidence instead of assumption.
func probeFrameCounters(rd *reader, dur time.Duration) {
	type state struct {
		first, last int32
		changes     int
		ok          bool
	}
	states := make([]state, len(frameCounterCandidates))
	for i := range frameCounterCandidates {
		v, err := rd.i32(frameCounterCandidates[i].addr)
		states[i] = state{first: v, last: v, ok: err == nil}
	}
	deadline := time.Now().Add(dur)
	for time.Now().Before(deadline) {
		for i := range frameCounterCandidates {
			v, err := rd.i32(frameCounterCandidates[i].addr)
			if err != nil {
				states[i].ok = false
				continue
			}
			if v != states[i].last {
				states[i].changes++
				states[i].last = v
			}
		}
		time.Sleep(2 * time.Millisecond)
	}
	fmt.Println("frame-counter probe (3s):")
	for i, c := range frameCounterCandidates {
		s := states[i]
		fmt.Printf("  %-14s 0x%08x  readable=%v  changes=%d  first=%d last=%d delta=%d\n",
			c.name, c.addr, s.ok, s.changes, s.first, s.last, s.last-s.first)
	}
}

type target struct {
	pid     uint32
	exePath string
}

func resolveTarget(pid uint32, exePath string) (target, error) {
	if pid != 0 {
		if exePath == "" {
			exePath = modulePathForPID(pid)
		}
		return target{pid: pid, exePath: exePath}, nil
	}
	if exePath == "" {
		return target{}, fmt.Errorf("need -pid or -exe")
	}
	p, err := findPIDByExe(exePath)
	if err != nil {
		return target{}, err
	}
	return target{pid: p, exePath: exePath}, nil
}

func findPIDByExe(exePath string) (uint32, error) {
	snap, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return 0, err
	}
	defer windows.CloseHandle(snap)
	want := strings.ToLower(filepath.Base(exePath))
	var e windows.ProcessEntry32
	e.Size = uint32(unsafe.Sizeof(e))
	if err := windows.Process32First(snap, &e); err != nil {
		return 0, err
	}
	for {
		name := strings.ToLower(windows.UTF16ToString(e.ExeFile[:]))
		if name == want {
			return e.ProcessID, nil
		}
		if err := windows.Process32Next(snap, &e); err != nil {
			break
		}
	}
	return 0, fmt.Errorf("process %q not running", want)
}

func modulePathForPID(pid uint32) string {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_INFORMATION, false, pid)
	if err != nil {
		return ""
	}
	defer windows.CloseHandle(h)
	var buf [windows.MAX_PATH]uint16
	n := uint32(len(buf))
	if err := windows.QueryFullProcessImageName(h, 0, &buf[0], &n); err != nil {
		return ""
	}
	return windows.UTF16ToString(buf[:n])
}

// moduleBase finds the load address of the target module inside pid's address
// space. Falls back to the static image base when enumeration is unavailable.
func moduleBase(pid uint32, exePath string) (uintptr, error) {
	snap, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPMODULE|windows.TH32CS_SNAPMODULE32, pid)
	if err != nil {
		return imageBase, nil
	}
	defer windows.CloseHandle(snap)
	want := strings.ToLower(filepath.Base(exePath))
	var m windows.ModuleEntry32
	m.Size = uint32(unsafe.Sizeof(m))
	if err := windows.Module32First(snap, &m); err != nil {
		return imageBase, nil
	}
	for {
		name := strings.ToLower(windows.UTF16ToString(m.ExePath[:]))
		if filepath.Base(name) == want {
			return uintptr(m.ModBaseAddr), nil
		}
		if err := windows.Module32Next(snap, &m); err != nil {
			break
		}
	}
	return imageBase, nil
}

func verifyHash(path string) error {
	if path == "" {
		return fmt.Errorf("cannot verify exe hash: path unknown (pass -exe or use -skip-hash)")
	}
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("hash: %w", err)
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return fmt.Errorf("hash: %w", err)
	}
	got := hex.EncodeToString(h.Sum(nil))
	if got != ExpectedHash {
		return fmt.Errorf("th10.exe hash mismatch:\n  got  %s\n  want %s", got, ExpectedHash)
	}
	return nil
}

// reader wraps read-only process memory access.
type reader struct {
	h windows.Handle
}

func (r *reader) bytes(addr uint32, n int) ([]byte, error) {
	buf := make([]byte, n)
	var read uintptr
	if err := windows.ReadProcessMemory(r.h, uintptr(addr), &buf[0], uintptr(n), &read); err != nil {
		return nil, fmt.Errorf("ReadProcessMemory 0x%x: %w", addr, err)
	}
	if int(read) != n {
		return nil, fmt.Errorf("short read at 0x%x: %d/%d", addr, read, n)
	}
	return buf, nil
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

func (r *reader) f32(addr uint32) (float32, error) {
	v, err := r.u32(addr)
	if err != nil {
		return 0, err
	}
	return math.Float32frombits(v), nil
}

func le32(b []byte) uint32 {
	return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
}

func le16(b []byte) uint16 { return uint16(b[0]) | uint16(b[1])<<8 }

func (r *reader) sampleFrame(index int) (*trace.Frame, error) {
	f := &trace.Frame{Index: index}

	score, _ := r.i32(addrScore)
	f.PScore = int64(score)
	life, _ := r.i32(addrLife)
	f.PLife = int(life)
	power, _ := r.i32(addrPower)
	f.PPower = int(power) * 5 // original raw 0.05 units -> project hundredths
	faith, _ := r.i32(addrFaith)
	f.PFaith = int64(faith)
	graze, _ := r.i32(addrGraze)
	f.PGraze = int(graze)

	pp, err := r.u32(addrPlayerPtr)
	if err == nil && pp != 0 {
		x, errX := r.f32(pp + addrPlayerX)
		y, errY := r.f32(pp + addrPlayerY)
		if errX == nil && errY == nil {
			f.PX, f.PY = float64(x), float64(y)
		}
		st, errSt := r.i32(pp + addrPlayerState)
		if errSt == nil {
			f.PState = int(st)
		}
	}

	// Enemies: walk the manager's list, main first.
	if em, err := r.u32(addrEnemyMgr); err == nil && em != 0 {
		node, _ := r.u32(em + enemyListHead)
		for n := 0; node != 0 && n < enemyMaxNodes; n++ {
			pair, err := r.bytes(node, 8)
			if err != nil {
				break
			}
			e, next := le32(pair[0:]), le32(pair[4:])
			node = next
			if e == 0 {
				continue
			}
			f.Enemies++
			if len(f.E) >= trace.MaxEnemyRows {
				continue
			}
			x, _ := r.f32(e + enemyPosX)
			y, _ := r.f32(e + enemyPosY)
			hp, _ := r.i32(e + enemyHP)
			flags, _ := r.u32(e + enemyFlags)
			f.E = append(f.E, trace.EnemyRow{
				X: float64(x), Y: float64(y), HP: int(hp), Active: flags&0x20000 == 0,
			})
		}
	}

	// Bullets: one read of the whole pool, then slot order.
	if bm, err := r.u32(addrBulletMgr); err == nil && bm != 0 {
		pool, err := r.bytes(bm+bulletArrayOff, bulletSlots*bulletStride)
		if err == nil {
			for i := 0; i < bulletSlots; i++ {
				b := pool[i*bulletStride:]
				if le16(b[bulletState:]) == 0 {
					continue
				}
				f.Bullets++
				if len(f.B) >= trace.MaxBulletRows {
					continue
				}
				f.B = append(f.B, trace.BulletRow{
					X:     float64(math.Float32frombits(le32(b[bulletPosX:]))),
					Y:     float64(math.Float32frombits(le32(b[bulletPosY:]))),
					Angle: float64(math.Float32frombits(le32(b[bulletAngle:]))),
					Speed: float64(math.Float32frombits(le32(b[bulletSpeed:]))),
					Type:  int(int16(le16(b[bulletType:]))),
					Color: int(int16(le16(b[bulletSprite:]))),
				})
			}
		}
	}
	return f, nil
}
