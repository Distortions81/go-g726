package g726

import (
	"bytes"
	"encoding/binary"
	"math"
	"testing"
)

func TestNewEncoderDecoderValidation(t *testing.T) {
	for _, bits := range []BitsPerSample{2, 3, 4, 5} {
		if _, err := NewEncoder(bits); err != nil {
			t.Fatalf("NewEncoder(%d) returned error: %v", bits, err)
		}
		if _, err := NewDecoder(bits); err != nil {
			t.Fatalf("NewDecoder(%d) returned error: %v", bits, err)
		}
	}

	for _, bits := range []BitsPerSample{-1, 0, 1, 6} {
		if _, err := NewEncoder(bits); err == nil {
			t.Fatalf("NewEncoder(%d) expected error", bits)
		}
		if _, err := NewDecoder(bits); err == nil {
			t.Fatalf("NewDecoder(%d) expected error", bits)
		}
	}
}

func TestEncodedSize(t *testing.T) {
	tests := []struct {
		bits    BitsPerSample
		pcmLen  int
		want    int
		wantErr bool
	}{
		{bits: 2, pcmLen: 0, want: 0},
		{bits: 2, pcmLen: 8, want: 1},
		{bits: 2, pcmLen: 10, wantErr: true},
		{bits: 3, pcmLen: 0, want: 0},
		{bits: 3, pcmLen: 16, want: 3},
		{bits: 3, pcmLen: 8, wantErr: true},
		{bits: 4, pcmLen: 0, want: 0},
		{bits: 4, pcmLen: 4, want: 1},
		{bits: 4, pcmLen: 6, wantErr: true},
		{bits: 5, pcmLen: 0, want: 0},
		{bits: 5, pcmLen: 16, want: 5},
		{bits: 5, pcmLen: 8, wantErr: true},
	}

	for _, tt := range tests {
		got, err := EncodedSize(tt.bits, tt.pcmLen)
		if tt.wantErr {
			if err == nil {
				t.Fatalf("EncodedSize(%d, %d) expected error", tt.bits, tt.pcmLen)
			}
			continue
		}
		if err != nil {
			t.Fatalf("EncodedSize(%d, %d) returned error: %v", tt.bits, tt.pcmLen, err)
		}
		if got != tt.want {
			t.Fatalf("EncodedSize(%d, %d) = %d, want %d", tt.bits, tt.pcmLen, got, tt.want)
		}
	}
}

func TestDecodedSize(t *testing.T) {
	tests := []struct {
		bits    BitsPerSample
		adpcm   int
		want    int
		wantErr bool
	}{
		{bits: 2, adpcm: 0, want: 0},
		{bits: 2, adpcm: 1, want: 8},
		{bits: 3, adpcm: 0, want: 0},
		{bits: 3, adpcm: 3, want: 16},
		{bits: 3, adpcm: 1, wantErr: true},
		{bits: 4, adpcm: 0, want: 0},
		{bits: 4, adpcm: 1, want: 4},
		{bits: 5, adpcm: 0, want: 0},
		{bits: 5, adpcm: 5, want: 16},
		{bits: 5, adpcm: 1, wantErr: true},
	}

	for _, tt := range tests {
		got, err := DecodedSize(tt.bits, tt.adpcm)
		if tt.wantErr {
			if err == nil {
				t.Fatalf("DecodedSize(%d, %d) expected error", tt.bits, tt.adpcm)
			}
			continue
		}
		if err != nil {
			t.Fatalf("DecodedSize(%d, %d) returned error: %v", tt.bits, tt.adpcm, err)
		}
		if got != tt.want {
			t.Fatalf("DecodedSize(%d, %d) = %d, want %d", tt.bits, tt.adpcm, got, tt.want)
		}
	}
}

func TestZeroLengthEncodeDecode(t *testing.T) {
	for _, bits := range []BitsPerSample{2, 3, 4, 5} {
		encoder, _ := NewEncoder(bits)
		decoder, _ := NewDecoder(bits)

		adpcm, err := encoder.Encode(nil)
		if err != nil || len(adpcm) != 0 {
			t.Fatalf("Encode nil for bits=%d => len=%d err=%v", bits, len(adpcm), err)
		}

		adpcm, err = encoder.EncodeBytes(nil)
		if err != nil || len(adpcm) != 0 {
			t.Fatalf("EncodeBytes nil for bits=%d => len=%d err=%v", bits, len(adpcm), err)
		}

		samples, err := decoder.Decode(nil)
		if err != nil || len(samples) != 0 {
			t.Fatalf("Decode nil for bits=%d => len=%d err=%v", bits, len(samples), err)
		}

		pcm, err := decoder.DecodeBytes(nil)
		if err != nil || len(pcm) != 0 {
			t.Fatalf("DecodeBytes nil for bits=%d => len=%d err=%v", bits, len(pcm), err)
		}
	}
}

func TestRoundTripAndHelpers(t *testing.T) {
	for _, bits := range []BitsPerSample{2, 3, 4, 5} {
		samples := fixtureSamples(bits, 32)
		pcm := int16sToBytes(samples)

		adpcmFromInts, err := Encode(bits, samples)
		if err != nil {
			t.Fatalf("Encode(%d) error: %v", bits, err)
		}

		adpcmFromBytes, err := EncodeBytes(bits, pcm)
		if err != nil {
			t.Fatalf("EncodeBytes(%d) error: %v", bits, err)
		}

		if !bytes.Equal(adpcmFromInts, adpcmFromBytes) {
			t.Fatalf("encoded payload mismatch for bits=%d", bits)
		}

		wantEncodedSize, _ := EncodedSize(bits, len(pcm))
		if len(adpcmFromInts) != wantEncodedSize {
			t.Fatalf("encoded size mismatch for bits=%d: got %d want %d", bits, len(adpcmFromInts), wantEncodedSize)
		}

		decodedSamples, err := Decode(bits, adpcmFromInts)
		if err != nil {
			t.Fatalf("Decode(%d) error: %v", bits, err)
		}

		decodedPCM, err := DecodeBytes(bits, adpcmFromInts)
		if err != nil {
			t.Fatalf("DecodeBytes(%d) error: %v", bits, err)
		}

		if len(decodedSamples) != len(samples) {
			t.Fatalf("decoded sample count mismatch for bits=%d: got %d want %d", bits, len(decodedSamples), len(samples))
		}
		if len(decodedPCM) != len(pcm) {
			t.Fatalf("decoded PCM byte count mismatch for bits=%d: got %d want %d", bits, len(decodedPCM), len(pcm))
		}
		if !bytes.Equal(decodedPCM, int16sToBytes(decodedSamples)) {
			t.Fatalf("decoded byte/sample mismatch for bits=%d", bits)
		}
	}
}

func TestStreamingMatchesSingleCall(t *testing.T) {
	for _, bits := range []BitsPerSample{2, 3, 4, 5} {
		samples := fixtureSamples(bits, 32)
		split := splitSamples(bits)

		fullEncoder, _ := NewEncoder(bits)
		fullADPCM, err := fullEncoder.Encode(samples)
		if err != nil {
			t.Fatalf("full encode for bits=%d returned error: %v", bits, err)
		}

		streamEncoder, _ := NewEncoder(bits)
		firstADPCM, err := streamEncoder.Encode(samples[:split])
		if err != nil {
			t.Fatalf("first stream encode for bits=%d returned error: %v", bits, err)
		}
		secondADPCM, err := streamEncoder.Encode(samples[split:])
		if err != nil {
			t.Fatalf("second stream encode for bits=%d returned error: %v", bits, err)
		}

		streamADPCM := append(firstADPCM, secondADPCM...)
		if !bytes.Equal(fullADPCM, streamADPCM) {
			t.Fatalf("streamed encode mismatch for bits=%d", bits)
		}

		fullDecoder, _ := NewDecoder(bits)
		fullPCM, err := fullDecoder.DecodeBytes(fullADPCM)
		if err != nil {
			t.Fatalf("full decode for bits=%d returned error: %v", bits, err)
		}

		streamDecoder, _ := NewDecoder(bits)
		firstPCM, err := streamDecoder.DecodeBytes(firstADPCM)
		if err != nil {
			t.Fatalf("first stream decode for bits=%d returned error: %v", bits, err)
		}
		secondPCM, err := streamDecoder.DecodeBytes(secondADPCM)
		if err != nil {
			t.Fatalf("second stream decode for bits=%d returned error: %v", bits, err)
		}

		streamPCM := append(firstPCM, secondPCM...)
		if !bytes.Equal(fullPCM, streamPCM) {
			t.Fatalf("streamed decode mismatch for bits=%d", bits)
		}
	}
}

func TestResetMatchesFreshState(t *testing.T) {
	for _, bits := range []BitsPerSample{2, 3, 4, 5} {
		samples := fixtureSamples(bits, 16)

		encoder, _ := NewEncoder(bits)
		if _, err := encoder.Encode(samples); err != nil {
			t.Fatalf("warmup encode for bits=%d returned error: %v", bits, err)
		}
		encoder.Reset()
		gotADPCM, err := encoder.Encode(samples)
		if err != nil {
			t.Fatalf("encode after reset for bits=%d returned error: %v", bits, err)
		}

		freshADPCM, err := Encode(bits, samples)
		if err != nil {
			t.Fatalf("fresh encode for bits=%d returned error: %v", bits, err)
		}
		if !bytes.Equal(gotADPCM, freshADPCM) {
			t.Fatalf("encoder reset mismatch for bits=%d", bits)
		}

		decoder, _ := NewDecoder(bits)
		if _, err := decoder.DecodeBytes(freshADPCM); err != nil {
			t.Fatalf("warmup decode for bits=%d returned error: %v", bits, err)
		}
		decoder.Reset()
		gotPCM, err := decoder.DecodeBytes(freshADPCM)
		if err != nil {
			t.Fatalf("decode after reset for bits=%d returned error: %v", bits, err)
		}

		freshPCM, err := DecodeBytes(bits, freshADPCM)
		if err != nil {
			t.Fatalf("fresh decode for bits=%d returned error: %v", bits, err)
		}
		if !bytes.Equal(gotPCM, freshPCM) {
			t.Fatalf("decoder reset mismatch for bits=%d", bits)
		}
	}
}

func TestEncodeBytesRejectsOddPCM(t *testing.T) {
	encoder, _ := NewEncoder(4)
	if _, err := encoder.EncodeBytes([]byte{1}); err == nil {
		t.Fatal("EncodeBytes expected error for odd PCM length")
	}
	if _, err := EncodeBytes(4, []byte{1}); err == nil {
		t.Fatal("package EncodeBytes expected error for odd PCM length")
	}
}

func TestEncodeRejectsMisalignedPCM(t *testing.T) {
	tests := []struct {
		bits BitsPerSample
		pcm  []byte
	}{
		{bits: 2, pcm: make([]byte, 2)},
		{bits: 3, pcm: make([]byte, 8)},
		{bits: 4, pcm: make([]byte, 2)},
		{bits: 5, pcm: make([]byte, 8)},
	}

	for _, tt := range tests {
		encoder, _ := NewEncoder(tt.bits)
		if _, err := encoder.EncodeBytes(tt.pcm); err == nil {
			t.Fatalf("EncodeBytes(%d, len=%d) expected alignment error", tt.bits, len(tt.pcm))
		}
		if _, err := EncodeBytes(tt.bits, tt.pcm); err == nil {
			t.Fatalf("package EncodeBytes(%d, len=%d) expected alignment error", tt.bits, len(tt.pcm))
		}
	}
}

func TestDecodeRejectsMisalignedADPCM(t *testing.T) {
	for _, bits := range []BitsPerSample{3, 5} {
		decoder, _ := NewDecoder(bits)
		if _, err := decoder.DecodeBytes([]byte{0x00}); err == nil {
			t.Fatalf("DecodeBytes(%d, len=1) expected alignment error", bits)
		}
		if _, err := DecodeBytes(bits, []byte{0x00}); err == nil {
			t.Fatalf("package DecodeBytes(%d, len=1) expected alignment error", bits)
		}
	}
}

func TestRegressionTwoBitStreamingState(t *testing.T) {
	samples := fixtureSamples(2, 8)

	fullEncoder, _ := NewEncoder(2)
	fullADPCM, err := fullEncoder.Encode(samples)
	if err != nil {
		t.Fatalf("full encode returned error: %v", err)
	}

	streamEncoder, _ := NewEncoder(2)
	first, err := streamEncoder.Encode(samples[:4])
	if err != nil {
		t.Fatalf("first stream encode returned error: %v", err)
	}
	second, err := streamEncoder.Encode(samples[4:])
	if err != nil {
		t.Fatalf("second stream encode returned error: %v", err)
	}

	if !bytes.Equal(fullADPCM, append(first, second...)) {
		t.Fatal("2 bits/sample encoder lost adaptive state across packed groups")
	}
}

func TestRegressionFiveBitEncodedSize(t *testing.T) {
	got, err := EncodedSize(5, 16)
	if err != nil {
		t.Fatalf("EncodedSize returned error: %v", err)
	}
	if got != 5 {
		t.Fatalf("EncodedSize(5, 16) = %d, want 5", got)
	}

	samples := fixtureSamples(5, 8)
	adpcm, err := Encode(5, samples)
	if err != nil {
		t.Fatalf("Encode returned error: %v", err)
	}
	if len(adpcm) != 5 {
		t.Fatalf("Encode(5, 8 samples) produced %d bytes, want 5", len(adpcm))
	}
}

func TestPackCodewordsLE(t *testing.T) {
	tests := []struct {
		name  string
		bits  uint
		codes []uint32
		want  []byte
	}{
		{name: "2-bit", bits: 2, codes: []uint32{0, 1, 2, 3}, want: []byte{0xE4}},
		{name: "3-bit", bits: 3, codes: []uint32{0, 1, 2, 3, 4, 5, 6, 7}, want: []byte{0x88, 0xC6, 0xFA}},
		{name: "4-bit", bits: 4, codes: []uint32{1, 2}, want: []byte{0x21}},
		{name: "5-bit", bits: 5, codes: []uint32{0, 1, 2, 3, 4, 5, 6, 7}, want: []byte{0x20, 0x88, 0x41, 0x8A, 0x39}},
	}

	for _, tt := range tests {
		dst := make([]byte, len(tt.want))
		n := packCodewordsLE(dst, tt.codes, tt.bits)
		if n != len(tt.want) {
			t.Fatalf("%s packed %d bytes, want %d", tt.name, n, len(tt.want))
		}
		if !bytes.Equal(dst, tt.want) {
			t.Fatalf("%s packed bytes = %x, want %x", tt.name, dst, tt.want)
		}
	}
}

func TestUnpackCodewordsLE(t *testing.T) {
	tests := []struct {
		name string
		bits uint
		src  []byte
		want []uint32
	}{
		{name: "2-bit", bits: 2, src: []byte{0xE4}, want: []uint32{0, 1, 2, 3}},
		{name: "3-bit", bits: 3, src: []byte{0x88, 0xC6, 0xFA}, want: []uint32{0, 1, 2, 3, 4, 5, 6, 7}},
		{name: "4-bit", bits: 4, src: []byte{0x21}, want: []uint32{1, 2}},
		{name: "5-bit", bits: 5, src: []byte{0x20, 0x88, 0x41, 0x8A, 0x39}, want: []uint32{0, 1, 2, 3, 4, 5, 6, 7}},
	}

	for _, tt := range tests {
		got := make([]uint32, len(tt.want))
		unpackCodewordsLE(tt.src, got, tt.bits)
		for i := range tt.want {
			if got[i] != tt.want[i] {
				t.Fatalf("%s unpacked codes[%d] = %d, want %d", tt.name, i, got[i], tt.want[i])
			}
		}
	}
}

func TestDecodeFiveBitReferenceFixture(t *testing.T) {
	adpcm := []byte{0xFF, 0xBD, 0xE7, 0x21, 0x84, 0xE2, 0x3D, 0xD6, 0x29, 0xC5}
	want := []int16{0, 188, 228, 280, 4, -244, -428, -684, -4, 824, 1792, 2832, 44, -3028, -4228, -2856}
	const tolerance = 32

	gotPCM, err := DecodeBytes(5, adpcm)
	if err != nil {
		t.Fatalf("DecodeBytes(5) returned error: %v", err)
	}
	if len(gotPCM) != len(want)*2 {
		t.Fatalf("DecodeBytes(5) returned %d bytes, want %d", len(gotPCM), len(want)*2)
	}

	for i, wantSample := range want {
		gotSample := int16(binary.LittleEndian.Uint16(gotPCM[i*2:]))
		diff := int(gotSample) - int(wantSample)
		if diff < 0 {
			diff = -diff
		}
		if diff > tolerance {
			t.Fatalf("decoded sample[%d] = %d, want %d within %d", i, gotSample, wantSample, tolerance)
		}
	}
}

func TestZeroRegionStaysSilentAtLargeStepSize(t *testing.T) {
	tests := []struct {
		name string
		sign int
		dqln int
		want int
	}{
		{name: "3-bit-positive", sign: 0, dqln: params3.reconstructTable[0], want: 0},
		{name: "3-bit-negative", sign: 1, dqln: params3.reconstructTable[7], want: -0x8000},
		{name: "5-bit-positive", sign: 0, dqln: params5.reconstructTable[0], want: 0},
		{name: "5-bit-negative", sign: 1, dqln: params5.reconstructTable[31], want: -0x8000},
	}

	for _, tt := range tests {
		if got := reconstruct(tt.sign, tt.dqln, 8192); got != tt.want {
			t.Fatalf("%s reconstruct(%d, %d, 8192) = %d, want %d", tt.name, tt.sign, tt.dqln, got, tt.want)
		}
	}
}

func TestFullScaleSineHasNoPopArtifacts(t *testing.T) {
	const (
		sampleRate  = 8000
		frequency   = 440.0
		amplitude   = 30000.0
		sampleCount = 800
	)

	src := make([]int16, sampleCount)
	for i := range src {
		src[i] = int16(math.Round(amplitude * math.Sin(2*math.Pi*frequency*float64(i)/sampleRate)))
	}

	for _, bits := range []BitsPerSample{2, 3, 4, 5} {
		adpcm, err := Encode(bits, src)
		if err != nil {
			t.Fatalf("Encode(%d) error: %v", bits, err)
		}

		decoded, err := Decode(bits, adpcm)
		if err != nil {
			t.Fatalf("Decode(%d) error: %v", bits, err)
		}

		var secondDerivativeSpikes int
		var signCrossingJumps int
		var worstCurvature int
		var worstCurvatureIndex int
		var worstJump int
		var worstJumpIndex int
		for i := 2; i < len(decoded); i++ {
			d1 := int(decoded[i]) - int(decoded[i-1])
			d0 := int(decoded[i-1]) - int(decoded[i-2])
			s1 := int(src[i]) - int(src[i-1])
			s0 := int(src[i-1]) - int(src[i-2])

			decodedCurvature := abs(d1 - d0)
			sourceCurvature := abs(s1 - s0)
			if decodedCurvature > worstCurvature {
				worstCurvature = decodedCurvature
				worstCurvatureIndex = i
			}
			if decodedCurvature > sourceCurvature*4+6000 {
				secondDerivativeSpikes++
			}

			jump := abs(int(decoded[i]) - int(decoded[i-1]))
			if jump > worstJump {
				worstJump = jump
				worstJumpIndex = i
			}
			if signInt16(decoded[i]) != signInt16(decoded[i-1]) && jump > 12000 {
				signCrossingJumps++
			}
		}

		if secondDerivativeSpikes != 0 || signCrossingJumps != 0 {
			t.Errorf(
				"bits=%d full-scale sine produced %d curvature spikes and %d large sign-crossing jumps; worst curvature at %d src=%v dec=%v; worst jump at %d src=%v dec=%v",
				bits,
				secondDerivativeSpikes,
				signCrossingJumps,
				worstCurvatureIndex,
				sampleWindow(src, worstCurvatureIndex, 4),
				sampleWindow(decoded, worstCurvatureIndex, 4),
				worstJumpIndex,
				sampleWindow(src, worstJumpIndex, 4),
				sampleWindow(decoded, worstJumpIndex, 4),
			)
		}
	}
}

func fixtureSamples(bits BitsPerSample, count int) []int16 {
	samples := make([]int16, count)
	for i := range samples {
		v := ((i * 173) + (int(bits) * 97)) % 4096
		samples[i] = int16((v - 2048) * 8)
	}
	return samples
}

func splitSamples(bits BitsPerSample) int {
	switch bits {
	case 2:
		return 16
	case 3:
		return 16
	case 4:
		return 14
	case 5:
		return 16
	default:
		return 0
	}
}

func signInt16(v int16) int {
	if v < 0 {
		return -1
	}
	if v > 0 {
		return 1
	}
	return 0
}

func sampleWindow(samples []int16, center, radius int) []int16 {
	start := center - radius
	if start < 0 {
		start = 0
	}
	end := center + radius + 1
	if end > len(samples) {
		end = len(samples)
	}
	return samples[start:end]
}
