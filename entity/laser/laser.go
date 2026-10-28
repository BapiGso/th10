// Package laser holds the enemy lasers ECL ops 412, 413 and 428-434 fire.
//
// This is the remake's first-pass model, moved here unchanged from the old
// float64 bullet pool so the bullet manager could be ported on its own. The
// binary's laser manager (FUN_0041c510 and the per-kind vtables FUN_0041c800
// and FUN_0041c850 walk) is not ported yet; positions are world coordinates.
package laser

import "math"

// Laser is one straight beam from (X, Y) along Angle.
type Laser struct {
	X, Y         float64
	Speed, Angle float64
	Length       float64
	Width        float64
	Color        int
	Active       bool
	Grazed       bool // grazed already (each laser grazes once)
	Age          int
	Lifetime     int   // >0: gone after this many frames
	ID           int32 // the handle ECL 413 stores and 414-419 look up
}

// Pool is a fixed set of laser slots.
type Pool struct {
	lasers []Laser
	lastID int32
}

// NewPool returns n free slots.
func NewPool(n int) *Pool {
	return &Pool{lasers: make([]Laser, n)}
}

// Fire starts a laser in the first free slot, or returns nil when the pool is
// full. A length or width of 0 or less takes 256 or 12.
func (p *Pool) Fire(x, y, speed, angle, length, width float64, color, lifetime int) *Laser {
	for i := range p.lasers {
		l := &p.lasers[i]
		if l.Active {
			continue
		}
		if length <= 0 {
			length = 256
		}
		if width <= 0 {
			width = 12
		}
		p.lastID++
		*l = Laser{
			X: x, Y: y, Speed: speed, Angle: angle,
			Length: length, Width: width, Color: color,
			Active: true, Lifetime: lifetime, ID: p.lastID,
		}
		return l
	}
	return nil
}

// Update moves every laser and drops the expired ones and those that left
// the playfield (left, top, right, bottom) by more than 96px plus their width.
func (p *Pool) Update(left, top, right, bottom float64) {
	for i := range p.lasers {
		l := &p.lasers[i]
		if !l.Active {
			continue
		}
		l.X += l.Speed * math.Cos(l.Angle)
		l.Y += l.Speed * math.Sin(l.Angle)
		l.Age++
		if l.Lifetime > 0 && l.Age >= l.Lifetime {
			l.Active = false
			continue
		}
		if !l.inBounds(left, top, right, bottom) {
			l.Active = false
		}
	}
}

// Find returns the active laser with handle id, or nil.
func (p *Pool) Find(id int32) *Laser {
	for i := range p.lasers {
		if l := &p.lasers[i]; l.Active && l.ID == id {
			return l
		}
	}
	return nil
}

// Each calls fn for every active laser.
func (p *Pool) Each(fn func(*Laser)) {
	for i := range p.lasers {
		if p.lasers[i].Active {
			fn(&p.lasers[i])
		}
	}
}

// ActiveCount is the number of active lasers.
func (p *Pool) ActiveCount() int {
	n := 0
	for i := range p.lasers {
		if p.lasers[i].Active {
			n++
		}
	}
	return n
}

// ClearAll removes every laser.
func (p *Pool) ClearAll() {
	for i := range p.lasers {
		p.lasers[i].Active = false
	}
}

// ClearCircle removes the lasers starting within r of (x, y), standing in for
// the laser half of ECL ops 420 and 421 (FUN_0041c800).
func (p *Pool) ClearCircle(x, y, r float64) {
	for i := range p.lasers {
		l := &p.lasers[i]
		if l.Active && math.Hypot(l.X-x, l.Y-y) <= r {
			l.Active = false
		}
	}
}

// End is the far end of the beam.
func (l *Laser) End() (x, y float64) {
	return l.X + math.Cos(l.Angle)*l.Length, l.Y + math.Sin(l.Angle)*l.Length
}

// HitPlayer reports whether a player circle of radius pr at (px, py) touches
// the beam.
func (l *Laser) HitPlayer(px, py, pr float64) bool {
	if !l.Active {
		return false
	}
	x2, y2 := l.End()
	return pointSegmentDistance(px, py, l.X, l.Y, x2, y2) <= pr+l.Width/2
}

// GrazePlayer reports a first graze: the player circle within 1.5 times the
// beam's half width.
func (l *Laser) GrazePlayer(px, py, pr float64) bool {
	if !l.Active || l.Grazed {
		return false
	}
	x2, y2 := l.End()
	if pointSegmentDistance(px, py, l.X, l.Y, x2, y2) <= pr+l.Width/2*1.5 {
		l.Grazed = true
		return true
	}
	return false
}

func (l *Laser) inBounds(left, top, right, bottom float64) bool {
	margin := 96 + math.Max(l.Width, 0)
	x2, y2 := l.End()
	minX, maxX := math.Min(l.X, x2), math.Max(l.X, x2)
	minY, maxY := math.Min(l.Y, y2), math.Max(l.Y, y2)
	return maxX >= left-margin && minX <= right+margin && maxY >= top-margin && minY <= bottom+margin
}

func pointSegmentDistance(px, py, ax, ay, bx, by float64) float64 {
	vx, vy := bx-ax, by-ay
	wx, wy := px-ax, py-ay
	len2 := vx*vx + vy*vy
	if len2 == 0 {
		return math.Hypot(px-ax, py-ay)
	}
	t := (wx*vx + wy*vy) / len2
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}
	return math.Hypot(px-(ax+t*vx), py-(ay+t*vy))
}
