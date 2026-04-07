package g726

type bits3Params struct {
	quantizerThresholds [3]int
	reconstructTable    [8]int
	scaleTable          [8]int
	stationarityTable   [8]int
}

var params3 bits3Params

func init() {
	params3.quantizerThresholds = [3]int{8, 218, 331}

	/* Maps 3-bit sample codes to reconstructed scale factor normalized log values. */
	params3.reconstructTable = [8]int{-2048, 135, 273, 373, 373, 273, 135, -2048}

	/* Maps 3-bit sample codes to scale-factor multiplier log values. */
	params3.scaleTable = [8]int{-128, 960, 4384, 18624, 18624, 4384, 960, -128}

	/*
	 * Maps 3-bit sample codes to values whose long and short
	 * term averages are computed and then compared to give an indication
	 * how stationary (steady state) the signal is.
	 */
	params3.stationarityTable = [8]int{0, 0x200, 0x400, 0xE00, 0xE00, 0x400, 0x200, 0}
}

func (state_ptr *codecState) encodeBits3(sl int) int {
	var (
		sezi  int
		sei   int
		sez   int
		se    int
		d     int
		y     int
		i     int
		dq    int
		sr    int
		dqsez int
	)

	sl >>= 2 /* sl of 14-bit dynamic range */

	sezi = state_ptr.predictor_zero()
	sez = sezi >> 1
	sei = sezi + state_ptr.predictor_pole()
	se = sei >> 1 /* se = estimated signal */

	d = sl - se /* d = estimation diff. */

	/* quantize prediction difference d */
	y = state_ptr.step_size()                                  /* quantizer step size */
	i = quantize(d, y, params3.quantizerThresholds[:])         /* i = ADPCM code */
	dq = reconstruct(i&4, int(params3.reconstructTable[i]), y) /* quantized diff. */

	sr = ifElse[int](dq < 0, se-(dq&0x3FFF), se+dq) /* reconstructed signal */

	dqsez = sr + sez - se /* pole prediction diff. */

	state_ptr.update(3, y, int(params3.scaleTable[i]), int(params3.stationarityTable[i]), dq, sr, dqsez)

	return i
}

func (state_ptr *codecState) decodeBits3(i int) int {
	var (
		sezi  int
		sez   int
		sei   int
		se    int
		y     int
		dq    int
		sr    int
		dqsez int
	)

	i &= 0x07 /* mask to get proper bits */
	sezi = state_ptr.predictor_zero()
	sez = sezi >> 1
	sei = sezi + state_ptr.predictor_pole()
	se = sei >> 1 /* se = estimated signal */

	y = state_ptr.step_size()                                     /* adaptive quantizer step size */
	dq = reconstruct(i&0x04, int(params3.reconstructTable[i]), y) /* unquantize pred diff */

	sr = ifElse[int](dq < 0, se-(dq&0x3FFF), se+dq) /* reconst. signal */

	dqsez = sr - se + sez /* pole prediction diff. */

	state_ptr.update(3, y, int(params3.scaleTable[i]), int(params3.stationarityTable[i]), dq, sr, dqsez)

	return sr << 2 /* sr was of 14-bit dynamic range */
}
