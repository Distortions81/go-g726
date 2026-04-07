package g726

import "testing"

func FuzzRoundTripInvariants(f *testing.F) {
	for _, bits := range []BitsPerSample{2, 3, 4, 5} {
		f.Add(int(bits), 1)
		f.Add(int(bits), 4)
		f.Add(int(bits), 9)
	}

	f.Fuzz(func(t *testing.T, bitsRaw int, frameCount int) {
		bits := BitsPerSample(bitsRaw)
		if err := validateBitsPerSample(bits); err != nil {
			t.Skip()
		}
		if frameCount < 0 {
			t.Skip()
		}
		if frameCount > 64 {
			frameCount = 64
		}

		samplesPerFrame := 2
		switch bits {
		case 2, 3, 5:
			samplesPerFrame = 8
		case 4:
			samplesPerFrame = 2
		}

		samples := fixtureSamples(bits, frameCount*samplesPerFrame)
		pcm := int16sToBytes(samples)

		adpcm, err := EncodeBytes(bits, pcm)
		if err != nil {
			t.Fatalf("EncodeBytes(%d, %d bytes) error: %v", bits, len(pcm), err)
		}

		wantEncoded, err := EncodedSize(bits, len(pcm))
		if err != nil {
			t.Fatalf("EncodedSize(%d, %d) error: %v", bits, len(pcm), err)
		}
		if len(adpcm) != wantEncoded {
			t.Fatalf("encoded size mismatch for bits=%d: got %d want %d", bits, len(adpcm), wantEncoded)
		}

		decodedPCM, err := DecodeBytes(bits, adpcm)
		if err != nil {
			t.Fatalf("DecodeBytes(%d, %d bytes) error: %v", bits, len(adpcm), err)
		}
		if len(decodedPCM) != len(pcm) {
			t.Fatalf("decoded size mismatch for bits=%d: got %d want %d", bits, len(decodedPCM), len(pcm))
		}
	})
}
