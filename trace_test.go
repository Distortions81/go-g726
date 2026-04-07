package g726

import (
	"fmt"
	"math"
	"testing"
)

func TestTraceLowBitSineWindows(t *testing.T) {
	const (
		sampleRate = 8000
		frequency  = 440.0
		amplitude  = 30000.0
	)

	src := make([]int16, 160)
	for i := range src {
		src[i] = int16(math.Round(amplitude * math.Sin(2*math.Pi*frequency*float64(i)/sampleRate)))
	}

	traceRange := map[BitsPerSample][2]int{
		2: {40, 46},
		3: {12, 20},
	}

	for _, bits := range []BitsPerSample{2, 3} {
		state := newCodecState(bits)
		window := traceRange[bits]
		t.Logf("bits=%d traceRange=%d..%d", bits, window[0], window[1])
		for i, sample := range src {
			var (
				code          int
				sezi          int
				sez           int
				sei           int
				se            int
				d             int
				y             int
				dq            int
				sr            int
				dqsez         int
				decodedSample int
			)

			sl := int(sample) >> 2
			sezi = state.predictor_zero()
			sez = sezi >> 1
			sei = sezi + state.predictor_pole()
			se = sei >> 1
			d = sl - se
			y = state.step_size()

			switch bits {
			case 2:
				code = quantize(d, y, params2.quantizerThresholds[:])
				if code == 3 && d >= 0 {
					code = 0
				}
				dq = reconstruct(code&2, params2.reconstructTable[code], y)
				sr = ifElse[int](dq < 0, se-(dq&0x3FFF), se+dq)
				dqsez = sr + sez - se
				state.update(2, y, params2.scaleTable[code], params2.stationarityTable[code], dq, sr, dqsez)
				decodedSample = clampPCM16(sr << 2)
			case 3:
				code = quantize(d, y, params3.quantizerThresholds[:])
				dq = reconstruct(code&4, params3.reconstructTable[code], y)
				sr = ifElse[int](dq < 0, se-(dq&0x3FFF), se+dq)
				dqsez = sr + sez - se
				state.update(3, y, params3.scaleTable[code], params3.stationarityTable[code], dq, sr, dqsez)
				decodedSample = clampPCM16(sr << 2)
			}

			if i >= window[0] && i <= window[1] {
				t.Logf(
					"bits=%d i=%d src=%d sl=%d se=%d d=%d y=%d code=%d dq=%d sr=%d dqsez=%d dec=%d a=%v b=%v pk=%v yu=%d yl=%d ap=%d td=%d",
					bits, i, sample, sl, se, d, y, code, dq, sr, dqsez, decodedSample,
					state.a, state.b, state.pk, state.yu, state.yl, state.ap, state.td,
				)
			}
		}
	}

	fmt.Print("")
}
