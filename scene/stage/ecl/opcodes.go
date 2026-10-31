package ecl

// TH10 opcode signature table, ported verbatim from thtk thecl10.c th10_fmts.
// Format chars: S=int32 f=float32 m=string x=encstring o=jump-offset t=jump-time
// D=typed call-arg(8B) H=variadic-typed *=repeat-next. Empty string = no params.
var th10Formats = map[uint16]string{
	0: "", 1: "", 10: "", 11: "m*D", 12: "ot", 13: "ot", 14: "ot", 15: "m*D",
	16: "mS*D", 17: "S", 21: "", 40: "S", 42: "S", 43: "S", 44: "f", 45: "f",
	50: "", 51: "", 52: "", 53: "", 54: "", 55: "", 56: "", 57: "", 58: "",
	59: "", 60: "", 61: "", 62: "", 63: "", 64: "", 65: "", 66: "", 67: "",
	68: "", 69: "", 70: "", 71: "", 72: "", 73: "", 74: "", 75: "", 76: "",
	77: "", 78: "S", 79: "", 80: "", 81: "ffff", 82: "f", 83: "S", 84: "",
	85: "", 86: "fff", 87: "fffff", 88: "", 89: "SSS",

	256: "mffSSS", 257: "mffSSS", 258: "S", 259: "SS", 260: "mffSSS", 261: "mffSSS",
	262: "SS", 263: "SS", 264: "SS", 265: "mffSSS", 266: "mffSSS", 267: "mffSSS",
	268: "mffSSS", 269: "S", 270: "mfffSSS", 271: "mfffSSS", 272: "SS", 273: "SSf",

	280: "ff", 281: "SSff", 282: "ff", 283: "SSff", 284: "ff", 285: "SSff",
	286: "ff", 287: "SSff", 288: "ffff", 289: "SSffff", 290: "ffff", 291: "SSffff",
	292: "SSf", 293: "SSf", 294: "", 295: "", 296: "fff", 297: "fff", 298: "ff", 299: "ff",

	320: "ff", 321: "ff", 322: "S", 323: "S", 324: "ffff", 325: "", 326: "",
	327: "SS", 328: "ff", 329: "", 330: "S", 331: "S", 332: "S", 333: "",
	334: "SSSm", 335: "S", 336: "S", 337: "SSS", 338: "S", 339: "", 340: "",
	341: "Sm", 342: "SSSx", 343: "", 344: "S", 345: "", 346: "f", 347: "SfS",
	348: "SSSx", 349: "ffff", 350: "ffffff", 351: "fff", 352: "SSSS", 353: "SSSSSS",
	354: "SSS", 355: "SSSSS", 356: "fffff", 357: "SSSx", 358: "SSSx", 359: "SSSx",
	360: "S", 361: "S", 362: "", 363: "", 364: "S", 365: "", 366: "SS", 367: "f",
	368: "SSSS",

	400: "S", 401: "S", 402: "SSS", 403: "Sff", 404: "Sff", 405: "Sff", 406: "SSS",
	407: "SS", 408: "SSS", 409: "SSSSSSff", 410: "", 411: "SS", 412: "SSffffSf",
	413: "SSSfffSSSSfS", 414: "Sff", 415: "Sff", 416: "Sf", 417: "Sf", 418: "Sf",
	419: "Sf", 420: "f", 421: "f", 422: "Sffffff", 423: "Sffffffffff", 424: "Sffff",
	425: "SSSSSSS", 426: "SSSSSSSSSSS", 427: "SSSSS", 428: "SSffSfSf",
	429: "SSSfffSSSSfS", 430: "Sff", 431: "SSffSfff", 432: "SSSfffSSSSfS",
	433: "SSffSfff", 434: "SSSfffSSSSfS", 435: "Sffffffff", 436: "SSSSSSSSS",
}

func th10Format(opcode uint16) string {
	if f, ok := th10Formats[opcode]; ok {
		return f
	}
	return "" // unknown: decodeParams falls back to all-int by blob length
}

// Control / system opcodes (thtk thecl.h TH10_INS_*, expr.c th10_expressions).
const (
	opRetBig       = 1  // delete owner + end task
	opRetNormal    = 10 // return from sub
	opCall         = 11 // m*D: call sub (sync)
	opGoto         = 12 // ot: unconditional jump
	opUnless       = 13 // ot: pop; if !v jump
	opIf           = 14 // ot: pop; if v jump
	opCallAsync    = 15 // m*D: spawn async task
	opCallAsyncID  = 16 // mS*D: spawn async task with slot id
	opKillAsyncID  = 17 // S
	opKillAllAsync = 21
	opStackAlloc   = 40 // S: local frame size in bytes
	opLoadI        = 42 // S: push int
	opAssignI      = 43 // S: pop -> int var
	opLoadF        = 44 // f: push float
	opAssignF      = 45 // f: pop -> float var

	opAddI = 50
	opAddF = 51
	opSubI = 52
	opSubF = 53
	opMulI = 54
	opMulF = 55
	opDivI = 56
	opDivF = 57
	opMod  = 58
	opEqI  = 59
	opEqF  = 60
	opNeI  = 61
	opNeF  = 62
	opLtI  = 63
	opLtF  = 64
	opLeI  = 65
	opLeF  = 66
	opGtI  = 67
	opGtF  = 68
	opGeI  = 69
	opGeF  = 70
	opNotI = 71
	opNotF = 72
	opOr   = 73
	opAnd  = 74
	opXor  = 75
	opBOr  = 76
	opBAnd = 77
	opDec  = 78 // S: var--
	opSin  = 79
	opCos  = 80
	opNegI = 84

	opPolar    = 81 // ffff: (outX, outY, angle, radius) polar->rect into two vars
	opValidRad = 82 // f: normalize angle var to (-pi,pi]
	opWait     = 83 // S: wait N frames
	opSqrt     = 88

	// Enemy / game opcodes (FUN_0040e770, 256-436).
	opEnmCreate   = 256 // mffSSS: (sub, x, y, hp, score, item) relative to the creator
	opEnmCreateA  = 257 // absolute
	opAnmSelect   = 258 // S: ANM file for the slot ops (0 = bullet.anm)
	opAnmSetSlot  = 259 // SS: (slot, script)
	opEnmCreateM  = 260 // relative, mirrored (flag 0x800)
	opEnmCreateAM = 261 // absolute, mirrored
	opAnmSetMain  = 262 // SS: (slot, script), also the direction base for 269
	opAnmPlay     = 263
	opAnmPlayAbs  = 264
	opAnmSetDir5  = 269 // S: slot plays the 262 script + 5
	opEnmCreate3D = 270 // mfffSSS: absolute with z (flag 0x40000)

	opSetPos        = 280 // ff
	opMovePos       = 281 // SSff: (time, mode, x, y)
	opSetPosB       = 282
	opMovePosB      = 283
	opSetVel        = 284 // ff: (angle, speed)
	opMoveVel       = 285 // SSff: (time, mode, angle, speed)
	opSetVelB       = 286
	opMoveVelB      = 287
	opSetCircle     = 288 // ffff: (angle, angular speed, radius, radial speed)
	opMoveCircle    = 289 // SSffff: (time, mode, ...as 288)
	opSetCircleB    = 290
	opMoveCircleB   = 291
	opMoveRand      = 292 // SSf: (time, mode, speed)
	opMoveRandB     = 293
	opPosToBoss     = 294 // (): layer A at boss 0
	opPosToBossB    = 295 // (): layer B at boss 0
	opPosAdd        = 296 // fff
	opPosAddB       = 297
	opCircleCentre  = 298 // ff: the layer's delta (centre), then A at boss 0
	opCircleCentreB = 299

	opSetHurtbox   = 320
	opSetHitbox    = 321
	opFlagSet      = 322
	opFlagClear    = 323
	opMoveLimit    = 324
	opMoveLimitR   = 325
	opDropClear    = 326
	opDropExtra    = 327
	opDropArea     = 328
	opDropItems    = 329
	opDropMain     = 330
	opSetHP        = 331
	opSetBoss      = 332
	opTimerReset   = 333
	opSetInterrupt = 334
	opSetInvuln    = 335
	opPlaySound    = 336
	opScreenShake  = 337
	opMSGOpen      = 338
	opMSGWait      = 339
	opBossWait     = 340
	opSetTimeout   = 341
	opSpellBg      = 342
	opSpellEnd     = 343
	opSetChapter   = 344
	opEnemySweep   = 345
	opLifeMarker   = 347
	opAnmRank      = 348
	opRankPickI    = 355
	opRankPickF    = 356
	opSpell        = 357
	opSpell3       = 359
	opStars        = 360
	opWaitRank     = 368 // SSSS: difficulty-indexed wait (Easy,Normal,Hard,Lunatic)

	// Bullet and laser ops (bullet_ops.go).
	opETNew       = 400 // S: reset a template
	opETFire      = 401 // S: fire a template
	opETSprite    = 402 // SSS: type, colour
	opETOffset    = 403 // Sff: fire offset from the enemy
	opETAngle     = 404 // Sff
	opETSpeed     = 405 // Sff
	opETCount     = 406 // SSS: per ring, rings
	opETAim       = 407 // SS: the angle mode
	opETSound     = 408 // SSS: fire sound, transform sound
	opETEx        = 409 // SSSSSSff: one transform entry
	opETClearAll  = 410 // rect clear with items, and every laser
	opETCopy      = 411 // SS: dst, src
	opLaserOnA    = 412 // SSffffff: line laser from slot 0's offset
	opLaserStOn   = 413 // vSSfffSSSSfS: infinite laser, its handle into a var
	opLaserPos    = 414 // Sff: 414-419 change the laser a handle names
	opLaserMove   = 415 // Sff
	opLaser416    = 416 // Sf
	opLaser417    = 417 // Sf
	opLaser418    = 418 // Sf
	opLaser419    = 419 // Sf
	opETCancel    = 420 // f: circle clear round the enemy, with items
	opETClear     = 421 // f: the same without items
	opETSpeedR3   = 422 // Sffffff: speeds by 3 rank tiers
	opETSpeedR5   = 423 // Sffffffffff: speeds by 5 rank tiers
	opETSpeedR2   = 424 // Sffff: speeds blended over rank
	opETCountR3   = 425 // SSSSSSS
	opETCountR5   = 426 // SSSSSSSSSSS
	opETCountR2   = 427 // SSSSS
	opLaserOnB    = 428 // 412 with laser flag 1 clear
	opLaserStB    = 429 // 413 with laser flag 2 clear
	opAimAt       = 430 // vff: the angle from (x, y) to the player
	opLaserOnAEx  = 431 // 412, 413, 428, 429 taking slot 0's transforms
	opLaserStOnEx = 432
	opLaserOnBEx  = 433
	opLaserStBEx  = 434
	opETSpeedDiff = 435 // Sffffffff: speeds by difficulty
	opETCountDiff = 436 // SSSSSSSSS: counts by difficulty
)
