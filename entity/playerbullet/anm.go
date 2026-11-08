package playerbullet

import (
	"th10/anm"
	"th10/zun"
)

// AttachAnm gives the shot records their original World/file association
// (P+0x10, FUN_00427e90). Nil disables only the animation paths.
func (p *Pool) AttachAnm(w *anm.World, env *anm.Env, file *anm.File) {
	p.world, p.env, p.file = w, env, file
}

func (p *Pool) anmAttached() bool { return p.world != nil && p.env != nil && p.file != nil }

func (p *Pool) spawnShot(b *Bullet) {
	if !p.anmAttached() {
		return
	}
	// FUN_00427e90, 0x42801a-0x42805f: start with layer 15 and AltRNG;
	// the shipped flight scripts set layer 8 in their first frame.
	b.VM = p.world.SpawnTail(p.file, b.shot.Script, 15, p.env)
	if b.VM.Flags&anm.FlagAutoRotate != 0 {
		b.VM.Rot[2] = float32(b.shot.Angle) // 0x4280bb; the update replaces this before drawing
		b.VM.Flags |= anm.FlagRotDirty
	}
	if b.shot.Kind == 3 {
		b.ExtraVM = p.world.SpawnTail(p.file, 0x10, 15, p.env) // 0x4280d0-0x428118
	}
}

// prep is the visual tail of FUN_00428280 (0x4283ca-0x428577).
func (p *Pool) prep(b *Bullet) {
	if !p.world.Contains(b.VM) {
		p.release(b) // 0x42841a-0x4284a5, also delete the second handle
		return
	}
	x, y := b.pos[0], b.pos[1]
	if b.Age >= 10 && b.shot.Kind != 3 {
		if sprite, ok := b.VM.File.Sprites[b.VM.Sprite]; ok {
			// R6: sprite record +0x34 is W, +0x30 is H (0x4284bc-0x4284da).
			w, h := float32(sprite.W*b.VM.Scale[0]), float32(sprite.H*b.VM.Scale[1])
			if x+w <= -192 || x-w >= 192 || y+h <= 0 || y-h >= 448 { // FUN_00428d70
				p.release(b)
				return
			}
		}
	}
	pos := [3]float32{x + 224, y + 16, b.Z}
	// Unlike the option loop's FUN_00449350, these are direct writes to
	// the two handles, not propagation through their child chains.
	b.VM.Offset = pos // 0x4284f7-0x428517
	if p.world.Contains(b.ExtraVM) {
		b.ExtraVM.Offset = pos // 0x428535-0x428555
	} else {
		b.ExtraVM = nil
	}
	if b.VM.Flags&anm.FlagAutoRotate != 0 {
		b.VM.Rot[2] = zun.AddAngle(float32(b.Angle), zun.HalfPi) // 0x428568
		b.VM.Flags |= anm.FlagRotDirty
	}
}

// hit is FUN_00428630's non-laser hit (0x4287d1-0x428832): replace the
// flight handle with a hit script, keep the old rotation, and continue the
// record at 1/8 speed until its World animation ends.
func (p *Pool) hit(b *Bullet) {
	if !p.anmAttached() || !p.world.Contains(b.VM) {
		p.release(b)
		return
	}
	rot := b.VM.Rot[2]
	p.world.Delete(b.VM)
	b.VM = p.world.SpawnTail(p.file, b.shot.HitScript, 15, p.env)
	b.VM.Rot[2], b.VM.Flags = rot, b.VM.Flags|anm.FlagRotDirty
	b.ScriptID = b.shot.HitScript
	b.Speed = f32(b.Speed * 0.125) // DAT_00470b90, 0x42881b
	b.Z = 0.1                      // 0x428824 = 0x3dcccccd
	b.state, b.Active = 2, false
}

func (p *Pool) release(b *Bullet) {
	if p.anmAttached() {
		p.world.Delete(b.VM)
		p.world.Delete(b.ExtraVM)
	}
	b.VM, b.ExtraVM, b.target = nil, nil, nil
	b.state, b.Active = 0, false
}
