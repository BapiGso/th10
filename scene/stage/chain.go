package stage

import "th10/supervisor"

// node is one node of th10.exe's draw chain (FUN_00449d40 over the list
// FUN_00449b70 builds; docs/research/m4_render_re.md §1.3): its priority,
// its callback's address, the supervisor work it starts with (enter) and
// what the stage draws in it (draw). Either is nil when there is none.
//
// Nodes not ported yet draw with Ebiten into the back buffer (legacy); see
// docs/research/m4_render.md §2.2 for which stage drawing sits in which
// node and why.
type node struct {
	prio  uint8
	fn    uint32
	enter func(sv *supervisor.Supervisor)
	draw  func(s *Stage, sv *supervisor.Supervisor)
}

// drawChain is the draw chain in priority order, every node the original
// registers whether or not it draws anything here.
var drawChain = []node{
	{prio: 0x01, fn: 0x420000, enter: (*supervisor.Supervisor).Start},
	{prio: 0x02, fn: 0x41fef0},
	{prio: 0x03, fn: 0x42d2f0},
	{prio: 0x06, fn: 0x40ba80},
	{prio: 0x07, fn: 0x403060, draw: (*Stage).nodeBackground}, // STD passes 0-7
	{prio: 0x09, fn: 0x4485f0, draw: layers(0)},               // layer 0
	{prio: 0x0a, fn: 0x403070, draw: layers(17, 18)},          // layers 17, 18, then STD passes 8-11
	{prio: 0x0b, fn: 0x448600, draw: layers(1)},               // layer 1
	{prio: 0x0d, fn: 0x448610, draw: layers(2)},               // layer 2
	{prio: 0x0e, fn: 0x409230},
	{prio: 0x0f, fn: 0x448620, enter: (*supervisor.Supervisor).EnterField, draw: layers(3)}, // layer 3
	{prio: 0x10, fn: 0x4486a0, draw: layers(4)},                                             // layer 4
	{prio: 0x11, fn: 0x4486b0, draw: layers(5)},                                             // layer 5
	{prio: 0x12, fn: 0x4486c0, draw: layers(6)},                                             // layer 6
	{prio: 0x13, fn: 0x4486d0, draw: layers(7)},                                             // layer 7
	{prio: 0x14, fn: 0x40d820, draw: (*Stage).nodeEnemies},                                  // enemies
	{prio: 0x15, fn: 0x4486e0, draw: layers(8)},                                             // layer 8
	{prio: 0x16, fn: 0x426510, draw: (*Stage).nodePlayer},                                   // player
	{prio: 0x17, fn: 0x4486f0, draw: layers(9)},                                             // layer 9
	{prio: 0x18, fn: 0x448700, draw: layers(10)},                                            // layer 10
	{prio: 0x19, fn: 0x41ba30, draw: (*Stage).nodeItems},                                    // items
	{prio: 0x1a, fn: 0x448710, draw: layers(11)},                                            // layer 11
	{prio: 0x1b, fn: 0x41c4e0, draw: (*Stage).nodeLasers},                                   // lasers
	{prio: 0x1c, fn: 0x448720, draw: layers(12)},                                            // layer 12
	{prio: 0x1d, fn: 0x4067a0, draw: (*Stage).nodeBullets},                                  // bullets
	{prio: 0x20, fn: 0x40b060},
	{prio: 0x21, fn: 0x448730, draw: layers(13)}, // layer 13
	{prio: 0x22, fn: 0x405850},
	{prio: 0x23, fn: 0x42b9b0},
	{prio: 0x24, fn: 0x448740, enter: (*supervisor.Supervisor).Unshake, draw: layers(14)}, // layer 14
	{prio: 0x25, fn: 0x409270},
	{prio: 0x26, fn: 0x401520},
	{prio: 0x28, fn: 0x4200c0},                                                            // returns 1, nothing else
	{prio: 0x29, fn: 0x448770, enter: (*supervisor.Supervisor).Enter2D, draw: layers(15)}, // layer 15
	{prio: 0x2a, fn: 0x4487c0, enter: (*supervisor.Supervisor).Enter2D, draw: layers(16)}, // layer 16
	{prio: 0x2b, fn: 0x415af0, draw: (*Stage).nodeGUI},                                    // GUI
	{prio: 0x2c, fn: 0x4198b0},
	{prio: 0x2d, fn: 0x448810, enter: (*supervisor.Supervisor).Enter2D}, // the layer-19 list
	{prio: 0x2e, fn: 0x422aa0, draw: (*Stage).nodePause},
	{prio: 0x30, fn: 0x401510},
	{prio: 0x32, fn: 0x4200d0, enter: (*supervisor.Supervisor).Finish},
}

// drawFrame is one frame's drawing on sv, FUN_00439390's draw section:
// BeginFrame, each node's enter then draw in chain order, EndFrame.
func (s *Stage) drawFrame(sv *supervisor.Supervisor) {
	sv.BeginFrame()
	for _, n := range drawChain {
		if n.enter != nil {
			n.enter(sv)
		}
		if n.draw != nil {
			n.draw(s, sv)
		}
	}
	sv.EndFrame()
}
