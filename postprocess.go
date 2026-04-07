package g726

import "fmt"

func Postprocess(bitsPerSample BitsPerSample, samples []int16) []int16 {
	if len(samples) < 3 {
		return append([]int16(nil), samples...)
	}

	out := append([]int16(nil), samples...)
	for pass := 0; pass < 2; pass++ {
		for i := 1; i < len(out)-1; i++ {
			prev := int(out[i-1])
			curr := int(out[i])
			next := int(out[i+1])

			interp := (prev + next) / 2
			median := median3(prev, curr, next)
			deviation := abs(curr - interp)
			left := abs(curr - prev)
			right := abs(curr - next)
			neighborSpan := abs(next - prev)

			if shouldReplaceWithMedian(bitsPerSample, prev, curr, next, left, right) {
				out[i] = int16(clampPCM16(median))
				continue
			}

			if shouldSmoothSample(bitsPerSample, deviation, left, right, neighborSpan) {
				out[i] = int16(clampPCM16((median + interp) / 2))
			}
		}
	}

	limitAbruptTransitions(bitsPerSample, out)

	return out
}

func PostprocessBytes(bitsPerSample BitsPerSample, pcmLE []byte) ([]byte, error) {
	if err := validateBitsPerSample(bitsPerSample); err != nil {
		return nil, err
	}
	if len(pcmLE)%2 != 0 {
		return nil, fmt.Errorf("pcm length must be even")
	}

	samples := bytesToInt16s(pcmLE)
	return int16sToBytes(Postprocess(bitsPerSample, samples)), nil
}

func shouldSmoothSample(bitsPerSample BitsPerSample, deviation, left, right, neighborSpan int) bool {
	baseDeviation := 4800
	baseStep := 5200

	switch bitsPerSample {
	case 2:
		baseDeviation = 3200
		baseStep = 4200
	case 3:
		baseDeviation = 4200
		baseStep = 4800
	case 4:
		baseDeviation = 5200
		baseStep = 5600
	case 5:
		baseDeviation = 5600
		baseStep = 6000
	}

	if deviation <= maxInt(baseDeviation, neighborSpan*2) {
		return false
	}

	if left <= baseStep && right <= baseStep {
		return false
	}

	return true
}

func shouldReplaceWithMedian(bitsPerSample BitsPerSample, prev, curr, next, left, right int) bool {
	outlierMargin := 5200
	stepThreshold := 6000

	switch bitsPerSample {
	case 2:
		outlierMargin = 2800
		stepThreshold = 4200
	case 3:
		outlierMargin = 3600
		stepThreshold = 5000
	case 4:
		outlierMargin = 4600
		stepThreshold = 5600
	case 5:
		outlierMargin = 5000
		stepThreshold = 6200
	}

	if left < stepThreshold || right < stepThreshold {
		return false
	}

	hi := maxInt(prev, next)
	lo := minInt(prev, next)
	return curr > hi+outlierMargin || curr < lo-outlierMargin
}

func limitAbruptTransitions(bitsPerSample BitsPerSample, samples []int16) {
	maxJump := 12000
	jerkThreshold := 16000

	switch bitsPerSample {
	case 2:
		maxJump = 9000
		jerkThreshold = 12000
	case 3:
		maxJump = 10000
		jerkThreshold = 14000
	case 4:
		maxJump = 12000
		jerkThreshold = 17000
	case 5:
		maxJump = 13000
		jerkThreshold = 18000
	}

	for pass := 0; pass < 2; pass++ {
		for i := 1; i < len(samples); i++ {
			prev := int(samples[i-1])
			curr := int(samples[i])
			delta := curr - prev

			limit := false
			if signInt16(samples[i]) != signInt16(samples[i-1]) && abs(delta) > maxJump {
				limit = true
			}
			if i >= 2 {
				prevDelta := int(samples[i-1]) - int(samples[i-2])
				if abs(delta-prevDelta) > jerkThreshold && abs(delta) > maxJump {
					limit = true
				}
			}

			if limit {
				samples[i] = int16(clampPCM16(prev + sign(delta)*maxJump))
			}
		}
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func median3(a, b, c int) int {
	if a > b {
		a, b = b, a
	}
	if b > c {
		b, c = c, b
	}
	if a > b {
		a, b = b, a
	}
	return b
}

func sign(v int) int {
	if v < 0 {
		return -1
	}
	if v > 0 {
		return 1
	}
	return 0
}
