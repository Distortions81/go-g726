package g726

type codecState struct {
	yl  int /* Locked or steady state step size multiplier. */
	yu  int /* Unlocked or non-steady state step size multiplier. */
	dms int /* Short term energy estimate. */
	dml int /* Long term energy estimate. */
	ap  int /* Linear weighting coefficient of 'yl' and 'yu'. */

	a  [2]int /* Coefficients of pole portion of prediction filter. */
	b  [6]int /* Coefficients of zero portion of prediction filter. */
	pk [2]int /* Signs of previous two samples of a partially
	 * reconstructed signal. */
	dq [6]int16 /* int here fails in newupdate on encode!
	 * Previous 6 samples of the quantized difference
	 * signal represented in an internal floating point
	 * format.
	 */
	sr [2]int /* Previous 2 samples of the quantized difference
	 * signal represented in an internal floating point
	 * format. */
	td int /* delayed tone detect, new in 1988 version */

	bitsPerSample BitsPerSample
}

func newCodecState(bitsPerSample BitsPerSample) codecState {
	state := codecState{}
	state.reset(bitsPerSample)
	return state
}

func (state_ptr *codecState) reset(bitsPerSample BitsPerSample) {
	*state_ptr = codecState{
		yl:            34816,
		yu:            544,
		dq:            [6]int16{32, 32, 32, 32, 32, 32},
		sr:            [2]int{32, 32},
		bitsPerSample: bitsPerSample,
	}
}

func (state_ptr *codecState) predictor_zero() int {
	var (
		i    int
		sezi int
	)

	sezi = fmult(state_ptr.b[0]>>2, int(state_ptr.dq[0]))
	for i = 1; i < 6; i++ { /* ACCUM */
		sezi += fmult(state_ptr.b[i]>>2, int(state_ptr.dq[i]))
	}

	return sezi
}

func (state_ptr *codecState) predictor_pole() int {
	return fmult(state_ptr.a[1]>>2, state_ptr.sr[1]) + fmult(state_ptr.a[0]>>2, state_ptr.sr[0])
}

func (state_ptr *codecState) step_size() int {
	var (
		y   int
		dif int
		al  int
	)

	if state_ptr.ap >= 256 {
		return state_ptr.yu
	} else {
		y = state_ptr.yl >> 6
		dif = state_ptr.yu - y
		al = state_ptr.ap >> 2
		if dif > 0 {
			y += (dif * al) >> 6
		} else if dif < 0 {
			y += (dif*al + 0x3F) >> 6
		}
		return y
	}
}

func quantize(d, y int, table []int) int {
	var (
		dqm  int
		exp  int
		mant int
		dl   int
		dln  int
		i    int
	)

	dqm = abs(d)
	exp = quan(dqm>>1, power2)

	mant = ((dqm << 7) >> exp) & 0x7F /* Fractional portion. */
	dl = (exp << 7) + mant

	dln = dl - (y >> 2)

	size := len(table)

	i = quan(dln, table)
	if d < 0 { /* take 1's complement of i */
		return (size << 1) + 1 - i
	} else if i == 0 { /* take 1's complement of 0 */
		return (size << 1) + 1 /* new in 1988 */
	} else {
		return i
	}
}

func reconstruct(sign, dqln, y int) int {
	var (
		dql int /* Log of 'dq' magnitude */
		dex int /* Integer part of log */
		dqt int
		dq  int /* Reconstructed difference signal sample */
	)

	dql = dqln + (y >> 2) /* ADDA */

	if dql < 0 {
		return ifElse[int](sign != 0, -0x8000, 0)
	} else { /* ANTILOG */
		dex = (dql >> 7) & 15
		dqt = 128 + (dql & 127)
		dq = (dqt << 7) >> (14 - dex)
		return ifElse(sign != 0, dq-0x8000, dq)
	}
}

func (state_ptr *codecState) update(code_size, y, wi, fi, dq, sr, dqsez int) {
	var (
		cnt                int
		mag, exp           int
		a2p                int
		a1ul               int
		pks1               int
		fa1                int
		tr                 int
		ylint, thr2, dqthr int
		ylfrac, thr1       int
		pk0                int
	)

	pk0 = ifElse[int](dqsez < 0, 1, 0) /* needed in updating predictor poles */

	mag = dq & 0x7FFF /* prediction difference magnitude */
	/* TRANS */
	ylint = state_ptr.yl >> 15                  /* exponent part of yl */
	ylfrac = (state_ptr.yl >> 10) & 0x1F        /* fractional part of yl */
	thr1 = (32 + ylfrac) << ylint               /* threshold */
	thr2 = ifElse[int](ylint > 9, 31<<10, thr1) /* limit thr2 to 31 << 10 */
	dqthr = (thr2 + (thr2 >> 1)) >> 1           /* dqthr = 0.75 * thr2 */

	if state_ptr.td == 0 { /* signal supposed voice */
		tr = 0
	} else if mag <= dqthr { /* supposed data, but small mag */
		tr = 0 /* treated as voice */
	} else { /* signal is data (modem) */
		tr = 1
	}

	/*
	 * Quantizer scale factor adaptation.
	 */

	/* FUNCTW & FILTD & DELAY */
	/* update non-steady state step size multiplier */
	state_ptr.yu = y + ((wi - y) >> 5)

	/* LIMB */
	if state_ptr.yu < 544 { /* 544 <= yu <= 5120 */
		state_ptr.yu = 544
	} else if state_ptr.yu > 5120 {
		state_ptr.yu = 5120
	}

	/* FILTE & DELAY */
	/* update steady state step size multiplier */
	t1 := state_ptr.yu + ((-state_ptr.yl) >> 6)
	t2 := state_ptr.yl + t1
	_ = t2
	state_ptr.yl += state_ptr.yu + ((-state_ptr.yl) >> 6)

	/*
	 * Adaptive predictor coefficients.
	 */
	if tr == 1 { /* reset a's and b's for modem signal */
		state_ptr.a[0] = 0
		state_ptr.a[1] = 0
		state_ptr.b[0] = 0
		state_ptr.b[1] = 0
		state_ptr.b[2] = 0
		state_ptr.b[3] = 0
		state_ptr.b[4] = 0
		state_ptr.b[5] = 0
		a2p = 0
	} else { /* update a's and b's */
		pks1 = pk0 ^ state_ptr.pk[0] /* UPA2 */

		/* update predictor pole a[1] */
		a2p = state_ptr.a[1] - (state_ptr.a[1] >> 7)
		if dqsez != 0 {
			fa1 = ifElse[int](pks1 != 0, state_ptr.a[0], -state_ptr.a[0])
			if fa1 < -8191 { /* a2p = function of fa1 */
				a2p -= 0x100
			} else if fa1 > 8191 {
				a2p += 0xFF
			} else {
				a2p += fa1 >> 5
			}

			if (pk0 ^ state_ptr.pk[1]) != 0 {
				/* LIMC */
				if a2p <= -12160 {
					a2p = -12288
				} else if a2p >= 12416 {
					a2p = 12288
				} else {
					a2p -= 0x80
				}
			} else if a2p <= -12416 {
				a2p = -12288
			} else if a2p >= 12160 {
				a2p = 12288
			} else {
				a2p += 0x80
			}
		}

		/* TRIGB & DELAY */
		state_ptr.a[1] = a2p

		/* UPA1 */
		/* update predictor pole a[0] */
		state_ptr.a[0] -= state_ptr.a[0] >> 8
		if dqsez != 0 {
			if pks1 == 0 {
				state_ptr.a[0] += 192
			} else {
				state_ptr.a[0] -= 192
			}
		}
		/* LIMD */
		a1ul = 15360 - a2p
		if state_ptr.a[0] < -a1ul {
			state_ptr.a[0] = -a1ul
		} else if state_ptr.a[0] > a1ul {
			state_ptr.a[0] = a1ul
		}

		/* UPB : update predictor zeros b[6] */
		for cnt = 0; cnt < 6; cnt++ {
			state_ptr.b[cnt] -= state_ptr.b[cnt] >> 8
			if (dq & 0x7FFF) != 0 { /* XOR */
				if (dq ^ int(state_ptr.dq[cnt])) >= 0 {
					state_ptr.b[cnt] += 128
				} else {
					state_ptr.b[cnt] -= 128
				}
			}
		}
	}

	for cnt = 5; cnt > 0; cnt-- {
		state_ptr.dq[cnt] = state_ptr.dq[cnt-1]
	}

	/* FLOAT A : convert dq[0] to 4-bit exp, 6-bit mantissa f.p. */
	if mag == 0 {
		u := uint16(0xFC20)
		state_ptr.dq[0] = ifElse[int16](dq >= 0, 0x20, int16(u))
	} else {
		exp = quan(mag, power2)
		state_ptr.dq[0] = ifElse[int16](dq >= 0, int16((exp<<6)+((mag<<6)>>exp)), int16((exp<<6)+((mag<<6)>>exp)-0x400))
	}

	state_ptr.sr[1] = state_ptr.sr[0]
	/* FLOAT B : convert sr to 4-bit exp., 6-bit mantissa f.p. */
	if sr == 0 {
		state_ptr.sr[0] = 0x20
	} else if sr > 0 {
		exp = quan(sr, power2)
		state_ptr.sr[0] = (exp << 6) + ((sr << 6) >> exp)
	} else if sr > -32768 {
		mag = -sr
		exp = quan(mag, power2)
		state_ptr.sr[0] = (exp << 6) + ((mag << 6) >> exp) - 0x400
	} else {
		state_ptr.sr[0] = 0xFC20
	}

	/* DELAY A */
	state_ptr.pk[1] = state_ptr.pk[0]
	state_ptr.pk[0] = pk0

	/* TONE */
	if tr == 1 { /* this sample has been treated as data */
		state_ptr.td = 0 /* next one will be treated as voice */
	} else if a2p < -11776 { /* small sample-to-sample correlation */
		state_ptr.td = 1 /* signal may be data */
	} else { /* signal is voice */
		state_ptr.td = 0
	}

	/*
	 * Adaptation speed control.
	 */
	state_ptr.dms += (fi - state_ptr.dms) >> 5        /* FILTA */
	state_ptr.dml += ((fi << 2) - state_ptr.dml) >> 7 /* FILTB */

	if tr == 1 {
		state_ptr.ap = 256
	} else if y < 1536 { /* SUBTC */
		state_ptr.ap += (0x200 - state_ptr.ap) >> 4
	} else if state_ptr.td == 1 {
		state_ptr.ap += (0x200 - state_ptr.ap) >> 4
	} else if abs((state_ptr.dms<<2)-state_ptr.dml) >= (state_ptr.dml >> 3) {
		state_ptr.ap += (0x200 - state_ptr.ap) >> 4
	} else {
		state_ptr.ap += (-state_ptr.ap) >> 4
	}
}

var power2 = []int{1, 2, 4, 8, 0x10, 0x20, 0x40, 0x80, 0x100, 0x200, 0x400, 0x800, 0x1000, 0x2000, 0x4000}

func quan(val int, table []int) int {
	for i := 0; i < len(table); i++ {
		if val < table[i] {
			return i
		}
	}
	return len(table)
}

func fmult(an, srn int) int {
	var (
		anmag   int
		anexp   int
		anmant  int
		wanexp  int
		wanmant int
		retval  int
	)

	anmag = ifElse[int](an > 0, an, (-an)&0x1FFF)
	anexp = quan(anmag, power2) - 6

	if anmag == 0 {
		anmant = 32
	} else if anexp >= 0 {
		anmant = anmag >> anexp
	} else {
		anmant = anmag << -anexp
	}
	wanexp = anexp + ((srn >> 6) & 0xF) - 13

	wanmant = (anmant*(srn&077) + 0x30) >> 4
	if wanexp >= 0 {
		retval = (wanmant << wanexp) & 0x7FFF
	} else {
		retval = wanmant >> -wanexp
	}

	return ifElse((an^srn) < 0, -retval, retval)
}

func ifElse[T any](b bool, t, f T) T {
	if b {
		return t
	} else {
		return f
	}
}

func signedReconstructDelta(dq, mask int) int {
	if dq < 0 {
		return -(dq & mask)
	}
	return dq
}

func clipPCMWord(sample int) int {
	if sample > 65535 {
		return 65535
	}
	if sample < -65535 {
		return -65535
	}
	return sample
}

func clampPCM16(sample int) int {
	if sample > 32767 {
		return 32767
	}
	if sample < -32768 {
		return -32768
	}
	return sample
}

type Compare interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 |
		~float32 | ~float64
}

func abs[T Compare](a T) T {
	if a < 0 {
		return -a
	} else {
		return a
	}
}
