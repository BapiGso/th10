// Package supervisortest is a recording supervisor.Device for tests.
package supervisortest

import (
	"fmt"
	"image"

	"th10/gfx"
)

// Rec records every device call as a line of text, in call order, and
// keeps the matrices it was given.
//
// The lines are:
//
//	tex <name>                  SetTexture; Names maps textures to names, nil is "nil"
//	filter <f>                  SetFilter
//	blend <b>                   SetBlend
//	draw <prim> <n>             DrawTL with n vertices
//	draw3d <prim> <n>           Draw3D with n vertices
//	transform <t>               SetTransform; the matrix goes to View, Proj, World or Tex0
//	viewport <x> <y> <w> <h> <minz> <maxz>
//	cull <c>
//	fog <mode> <color hex> <start> <end> <range>
//	fogenable <on>
//	address <a>
//	zenable <on>
//	zwrite <on>
//	zfunc <f>
//	alphatest <on> <ref>
//	clear <flags> <color hex> <z> <rects>
//	flush
type Rec struct {
	Calls []string
	Names map[*gfx.Texture]string

	View, Proj, World, Tex0 gfx.Mat4
}

func (d *Rec) add(format string, a ...any) { d.Calls = append(d.Calls, fmt.Sprintf(format, a...)) }

// Take returns the calls since the last Take.
func (d *Rec) Take() []string {
	c := d.Calls
	d.Calls = nil
	return c
}

func (d *Rec) name(t *gfx.Texture) string {
	if t == nil {
		return "nil"
	}
	if n, ok := d.Names[t]; ok {
		return n
	}
	return "?"
}

func (d *Rec) SetTexture(t *gfx.Texture) { d.add("tex %s", d.name(t)) }
func (d *Rec) SetFilter(f gfx.Filter)    { d.add("filter %d", f) }
func (d *Rec) SetBlend(b gfx.BlendMode)  { d.add("blend %d", b) }
func (d *Rec) DrawTL(p gfx.Primitive, vs []gfx.TLVertex) {
	d.add("draw %d %d", p, len(vs))
}
func (d *Rec) Draw3D(p gfx.Primitive, vs []gfx.Vertex3D) {
	d.add("draw3d %d %d", p, len(vs))
}

func (d *Rec) SetTransform(t gfx.TransformState, m gfx.Mat4) {
	switch t {
	case gfx.View:
		d.View = m
	case gfx.Projection:
		d.Proj = m
	case gfx.World:
		d.World = m
	case gfx.Texture0:
		d.Tex0 = m
	}
	d.add("transform %d", t)
}

func (d *Rec) SetViewport(v gfx.Viewport) {
	d.add("viewport %g %g %g %g %g %g", v.X, v.Y, v.W, v.H, v.MinZ, v.MaxZ)
}
func (d *Rec) SetCull(c gfx.Cull) { d.add("cull %d", c) }
func (d *Rec) SetFog(f gfx.Fog) {
	d.add("fog %d %08x %g %g %v", f.Mode, f.Color, f.Start, f.End, f.Range)
}
func (d *Rec) SetFogEnable(on bool)     { d.add("fogenable %v", on) }
func (d *Rec) SetAddress(a gfx.Address) { d.add("address %d", a) }
func (d *Rec) SetZEnable(on bool)       { d.add("zenable %v", on) }
func (d *Rec) SetZWrite(on bool)        { d.add("zwrite %v", on) }
func (d *Rec) SetZFunc(f gfx.CmpFunc)   { d.add("zfunc %d", f) }
func (d *Rec) SetAlphaTest(on bool, ref uint8) {
	d.add("alphatest %v %d", on, ref)
}
func (d *Rec) Clear(f gfx.ClearFlags, c gfx.Color, z float32, rects ...image.Rectangle) {
	d.add("clear %d %08x %g %v", f, c, z, rects)
}
func (d *Rec) Flush() { d.add("flush") }
