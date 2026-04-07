package g726

type bits4Params struct {
	quantizerThresholds [7]int
	reconstructTable    [16]int
	scaleTable          [16]int
	stationarityTable   [16]int
}

var params4 bits4Params

func init() {
	params4.quantizerThresholds = [7]int{-124, 80, 178, 246, 300, 349, 400}
	/* Maps 4-bit sample codes to reconstructed scale factor normalized log values. */
	params4.reconstructTable = [16]int{-2048, 4, 135, 213, 273, 323, 373, 425,
		425, 373, 323, 273, 213, 135, 4, -2048}

	/* Maps 4-bit sample codes to scale-factor multiplier log values. */
	params4.scaleTable = [16]int{-12, 18, 41, 64, 112, 198, 355, 1122,
		1122, 355, 198, 112, 64, 41, 18, -12}
	/*
	 * Maps 4-bit sample codes to values whose long and short
	 * term averages are computed and then compared to give an indication
	 * how stationary (steady state) the signal is.
	 */
	params4.stationarityTable = [16]int{0, 0, 0, 0x200, 0x200, 0x200, 0x600, 0xE00,
		0xE00, 0x600, 0x200, 0x200, 0x200, 0, 0, 0}
}

func (state_ptr *codecState) encodeBits4(sl int) int {
	var (
		sezi  int
		sez   int
		se    int
		d     int
		y     int
		i     int
		dq    int
		sr    int
		dqsez int
	)

	sl >>= 2 /* 14-bit dynamic range */

	sezi = state_ptr.predictor_zero()
	sez = sezi >> 1
	se = (sezi + state_ptr.predictor_pole()) >> 1 /* estimated signal */

	d = sl - se /* estimation difference */

	/* quantize the prediction difference */
	y = state_ptr.step_size()                          /* quantizer step size */
	i = quantize(d, y, params4.quantizerThresholds[:]) /* i = ADPCM code */

	dq = reconstruct(i&8, params4.reconstructTable[i], y) /* quantized est diff */

	sr = ifElse[int](dq < 0, se-(dq&0x3FFF), se+dq) /* reconst. signal */

	dqsez = sr + sez - se /* pole prediction diff. */

	t0 := int(params4.scaleTable[i])
	t1 := t0 << 5
	t2 := int(t1)
	_ = t2
	state_ptr.update(4, y, params4.scaleTable[i]<<5, params4.stationarityTable[i], dq, sr, dqsez)

	return i
}

func (state_ptr *codecState) decodeBits4(i int) int {
	var (
		sezi  int
		sez   int
		sei   int
		se    int
		y     int
		dq    int
		sr    int
		dqsez int
		lino  int
	)

	i &= 0x0f /* mask to get proper bits */
	sezi = state_ptr.predictor_zero()
	sez = sezi >> 1
	sei = sezi + state_ptr.predictor_pole()
	se = sei >> 1 /* se = estimated signal */

	y = state_ptr.step_size() /* dynamic quantizer step size */

	dq = reconstruct(i&0x08, params4.reconstructTable[i], y) /* quantized diff. */

	sr = ifElse[int](dq < 0, se-(dq&0x3FFF), se+dq) /* reconst. signal */

	dqsez = sr - se + sez /* pole prediction diff. */

	state_ptr.update(4, y, params4.scaleTable[i]<<5, params4.stationarityTable[i], dq, sr, dqsez)

	lino = sr << 2 /* this seems to overflow a short*/
	lino = ifElse[int](lino > 32767, 32767, lino)
	lino = ifElse[int](lino < -32768, -32768, lino)

	return lino //(sr << 2);	/* sr was 14-bit dynamic range */
}
