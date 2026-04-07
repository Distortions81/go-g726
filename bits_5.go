package g726

type bits5Params struct {
	quantizerThresholds [15]int
	reconstructTable    [32]int
	scaleTable          [32]int
	stationarityTable   [32]int
}

var params5 bits5Params

func init() {
	params5.quantizerThresholds = [15]int{-122, -16, 68, 139, 198, 250, 298, 339,
		378, 413, 445, 475, 502, 528, 553}
	/* Maps 5-bit sample codes to reconstructed scale factor normalized log values. */
	params5.reconstructTable = [32]int{-2048, -66, 28, 104, 169, 224, 274, 318,
		358, 395, 429, 459, 488, 514, 539, 566,
		566, 539, 514, 488, 459, 429, 395, 358,
		318, 274, 224, 169, 104, 28, -66, -2048}

	/* Maps 5-bit sample codes to scale-factor multiplier log values. */
	params5.scaleTable = [32]int{448, 448, 768, 1248, 1280, 1312, 1856, 3200,
		4512, 5728, 7008, 8960, 11456, 14080, 16928, 22272,
		22272, 16928, 14080, 11456, 8960, 7008, 5728, 4512,
		3200, 1856, 1312, 1280, 1248, 768, 448, 448}

	/*
	 * Maps 5-bit sample codes to values whose long and short
	 * term averages are computed and then compared to give an indication
	 * how stationary (steady state) the signal is.
	 */
	params5.stationarityTable = [32]int{0, 0, 0, 0, 0, 0x200, 0x200, 0x200,
		0x200, 0x200, 0x400, 0x600, 0x800, 0xA00, 0xC00, 0xC00,
		0xC00, 0xC00, 0xA00, 0x800, 0x600, 0x400, 0x200, 0x200,
		0x200, 0x200, 0x200, 0, 0, 0, 0, 0}

}

func (state_ptr *codecState) encodeBits5(sl int) int {

	var (
		sezi  int
		sez   int
		sei   int
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

	d = sl - se /* d = estimation difference */

	/* quantize prediction difference */
	y = state_ptr.step_size()                          /* adaptive quantizer step size */
	i = quantize(d, y, params5.quantizerThresholds[:]) /* i = ADPCM code */

	dq = reconstruct(i&0x10, int(params5.reconstructTable[i]), y) /* quantized diff */

	sr = ifElse[int](dq < 0, se-(dq&0x7FFF), se+dq) /* reconstructed signal */

	dqsez = sr + sez - se /* dqsez = pole prediction diff. */

	state_ptr.update(5, y, int(params5.scaleTable[i]), int(params5.stationarityTable[i]), dq, sr, dqsez)

	return i
}

func (state_ptr *codecState) decodeBits5(i int) int {
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

	i &= 0x1f /* mask to get proper bits */
	sezi = state_ptr.predictor_zero()
	sez = sezi >> 1
	sei = sezi + state_ptr.predictor_pole()
	se = sei >> 1 /* se = estimated signal */

	y = state_ptr.step_size()                                     /* adaptive quantizer step size */
	dq = reconstruct(i&0x10, int(params5.reconstructTable[i]), y) /* estimation diff. */

	sr = ifElse[int](dq < 0, se-(dq&0x7FFF), se+dq) /* reconst. signal */

	dqsez = sr - se + sez /* pole prediction diff. */

	state_ptr.update(5, y, int(params5.scaleTable[i]), int(params5.stationarityTable[i]), dq, sr, dqsez)

	return sr << 2 /* sr was of 14-bit dynamic range */
}
