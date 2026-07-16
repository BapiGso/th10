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
	268: "mffSSS", 269: "S", 270: "mffSSSS", 271: "mffSSSS", 272: "SS", 273: "SSf",

	280: "ff", 281: "SSff", 282: "ff", 283: "SSfS", 284: "ff", 285: "SSff",
	286: "ff", 287: "SSff", 288: "ffff", 290: "ffff", 291: "SSfffS", 292: "SSf",
	293: "SSf", 294: "", 295: "", 296: "SSf", 297: "ff", 298: "ff", 299: "ff",

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
	opEnmCreate     = 256
	opEnmCreateM    = 257
	opAnmSelect     = 258
	opAnmSetSprite  = 259
	opEnmCreateAbs  = 260
	opEnmCreateAbsM = 261
	opAnmSetMain    = 262
	opAnmPlay       = 263
	opAnmPlayAbs    = 264
	opAnmSelectPlay = 269

	opSetPos       = 280
	opMovePosTime  = 281
	opSetPosB      = 282
	opMovePosTimeB = 283
	opSetVel       = 284
	opMoveVelTime  = 285
	opSetVelB      = 286
	opMoveVelTimeB = 287
	opMoveCircle   = 288
	opMoveCircleB  = 290
	opMoveRand     = 292
	opMoveRandB    = 293
	opMoveStop     = 294 // (): halt motion on layer A
	opMoveAdd      = 296
	opMoveAddB     = 298
	opMoveVel299   = 299 // ff: set layer-B heading/speed instantly

	opSetHurtbox  = 320
	opSetHitbox   = 321
	opFlagSet     = 322
	opFlagClear   = 323
	opMoveLimit   = 324
	opMoveLimitR  = 325
	opDropClear   = 326
	opDropExtra   = 327
	opDropArea    = 328
	opDropItems   = 329
	opDropMain    = 330
	opSetHP       = 331
	opSetBoss     = 332
	opTimerReset  = 333
	opDelayedCall = 334
	opSetInvuln   = 335
	opPlaySound   = 336
	opScreenShake = 337
	opDialogRead  = 338
	opDialogWait  = 339
	opDeathWait   = 340
	opSetTimeout  = 341
	opSpellBg     = 342
	opSpellEnd    = 343
	opSetChapter  = 344
	opLifeMarker  = 347
	opAnmRank     = 348
	opRankPickI   = 355
	opRankPickF   = 356
	opSpell       = 357
	opSpell3      = 359
	opStars       = 360
	opWaitRank    = 368 // SSSS: difficulty-indexed wait (Easy,Normal,Hard,Lunatic)

	opETNew        = 400
	opETFire       = 401
	opETSprite     = 402
	opETOffset     = 403
	opETAngle      = 404
	opETSpeed      = 405
	opETCount      = 406
	opETAim        = 407
	opETSound      = 408
	opETEx         = 409
	opETClearAll   = 410
	opETCopy       = 411
	opLaserOnA     = 412
	opLaserStOn    = 413
	opLaserOnB     = 428 // SSffSfSf: boss laser variant
	opLaserStC     = 431 // SSffSfff: Extra-stage boss laser variant
	opLaserStB     = 433 // SSffSfff: boss laser variant
	opETCancel     = 420
	opETClear      = 421
	opETCountR3    = 425
	opETCountR5    = 426
	opETCountR2    = 427
	opETSpeedRank4 = 435 // Sffffffff: slot + speed[4 diff] + speed2[4 diff]
	opETCountRank4 = 436 // SSSSSSSSS: slot + ways[4 diff] + stacks[4 diff]
)
