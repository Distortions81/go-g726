package g726

type bits2Params struct {
	quantizerThresholds [1]int
	reconstructTable    [4]int
	scaleTable          [4]int
	stationarityTable   [4]int
}

var params2 bits2Params

func init() {
	params2.quantizerThresholds = [1]int{260}

	/*
	 * Maps 2-bit sample codes to reconstructed scale factor normalized log
	 * magnitude values.
	 */
	params2.reconstructTable = [4]int{116, 365, 365, 116}

	/* Maps 2-bit sample codes to the scale-factor multiplier log.
	 *
	 * The reference table is scaled by 32 to match the update routine.
	 */
	params2.scaleTable = [4]int{-704, 14048, 14048, -704}

	/*
	 * Maps 2-bit sample codes to values whose long and short
	 * term averages are computed and then compared to give an indication
	 * how stationary (steady state) the signal is.
	 */

	/* Comes from FUNCTF */
	params2.stationarityTable = [4]int{0, 0xE00, 0xE00, 0}
}

func (state_ptr *codecState) encodeBits2(sl int) int {
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

	d = sl - se /* d = estimation diff. */

	/* quantize prediction difference d */
	y = state_ptr.step_size()                          /* quantizer step size */
	i = quantize(d, y, params2.quantizerThresholds[:]) /* i = ADPCM code */

	/* Since quantize() only produces a three level output
	 * (1, 2, or 3), we must create the fourth one on our own
	 */
	if i == 3 { /* i code for the zero region */
		if d >= 0 { /* If d > 0, i=3 isn't right... */
			i = 0
		}
	}

	dq = reconstruct(i&2, int(params2.reconstructTable[i]), y) /* quantized diff. */

	sr = int(int16(se + signedReconstructDelta(dq, 0x3FFF))) /* reconstructed signal */

	dqsez = sr + sez - se /* pole prediction diff. */

	state_ptr.update(2, y, int(params2.scaleTable[i]), int(params2.stationarityTable[i]), dq, sr, dqsez)

	return i
}

func (state_ptr *codecState) decodeBits2(i int) int {
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

	i &= 0x03 /* mask to get proper bits */
	sezi = state_ptr.predictor_zero()
	sez = sezi >> 1
	sei = sezi + state_ptr.predictor_pole()
	se = sei >> 1 /* se = estimated signal */

	y = state_ptr.step_size()                                     /* adaptive quantizer step size */
	dq = reconstruct(i&0x02, int(params2.reconstructTable[i]), y) /* unquantize pred diff */

	sr = int(int16(se + signedReconstructDelta(dq, 0x3FFF))) /* reconst. signal */

	dqsez = sr - se + sez /* pole prediction diff. */

	state_ptr.update(2, y, int(params2.scaleTable[i]), int(params2.stationarityTable[i]), dq, sr, dqsez)

	return clampPCM16(sr << 2) /* sr was of 14-bit dynamic range */
}
