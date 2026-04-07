package g726

import (
	"encoding/binary"
	"fmt"
)

type BitsPerSample int

type Encoder struct {
	state codecState
}

type Decoder struct {
	state codecState
}

func NewEncoder(bitsPerSample BitsPerSample) (*Encoder, error) {
	if err := validateBitsPerSample(bitsPerSample); err != nil {
		return nil, err
	}

	return &Encoder{state: newCodecState(bitsPerSample)}, nil
}

func NewDecoder(bitsPerSample BitsPerSample) (*Decoder, error) {
	if err := validateBitsPerSample(bitsPerSample); err != nil {
		return nil, err
	}

	return &Decoder{state: newCodecState(bitsPerSample)}, nil
}

func (e *Encoder) Reset() {
	e.state.reset(e.state.bitsPerSample)
}

func (d *Decoder) Reset() {
	d.state.reset(d.state.bitsPerSample)
}

func Encode(bitsPerSample BitsPerSample, samples []int16) ([]byte, error) {
	encoder, err := NewEncoder(bitsPerSample)
	if err != nil {
		return nil, err
	}
	return encoder.Encode(samples)
}

func EncodeBytes(bitsPerSample BitsPerSample, pcmLE []byte) ([]byte, error) {
	encoder, err := NewEncoder(bitsPerSample)
	if err != nil {
		return nil, err
	}
	return encoder.EncodeBytes(pcmLE)
}

func Decode(bitsPerSample BitsPerSample, data []byte) ([]int16, error) {
	decoder, err := NewDecoder(bitsPerSample)
	if err != nil {
		return nil, err
	}
	return decoder.Decode(data)
}

func DecodePostProcessed(bitsPerSample BitsPerSample, data []byte) ([]int16, error) {
	decoder, err := NewDecoder(bitsPerSample)
	if err != nil {
		return nil, err
	}
	return decoder.DecodePostProcessed(data)
}

func DecodeBytes(bitsPerSample BitsPerSample, data []byte) ([]byte, error) {
	decoder, err := NewDecoder(bitsPerSample)
	if err != nil {
		return nil, err
	}
	return decoder.DecodeBytes(data)
}

func DecodeBytesPostProcessed(bitsPerSample BitsPerSample, data []byte) ([]byte, error) {
	decoder, err := NewDecoder(bitsPerSample)
	if err != nil {
		return nil, err
	}
	return decoder.DecodeBytesPostProcessed(data)
}

func EncodedSize(bitsPerSample BitsPerSample, pcmBytes int) (int, error) {
	if err := validateBitsPerSample(bitsPerSample); err != nil {
		return -1, err
	}
	if pcmBytes < 0 {
		return -1, fmt.Errorf("pcm length must be non-negative")
	}

	switch bitsPerSample {
	case BitsPerSample(2):
		if pcmBytes%8 != 0 {
			return -1, fmt.Errorf("pcm length must be a multiple of 8 bytes for 2 bits/sample encoding")
		}
		return pcmBytes / 8, nil
	case BitsPerSample(3):
		if pcmBytes%16 != 0 {
			return -1, fmt.Errorf("pcm length must be a multiple of 16 bytes for 3 bits/sample encoding")
		}
		return pcmBytes / 16 * 3, nil
	case BitsPerSample(4):
		if pcmBytes%4 != 0 {
			return -1, fmt.Errorf("pcm length must be a multiple of 4 bytes for 4 bits/sample encoding")
		}
		return pcmBytes / 4, nil
	case BitsPerSample(5):
		if pcmBytes%16 != 0 {
			return -1, fmt.Errorf("pcm length must be a multiple of 16 bytes for 5 bits/sample encoding")
		}
		return pcmBytes / 16 * 5, nil
	default:
		return -1, fmt.Errorf("invalid bits per sample: %d", bitsPerSample)
	}
}

func DecodedSize(bitsPerSample BitsPerSample, adpcmBytes int) (int, error) {
	if err := validateBitsPerSample(bitsPerSample); err != nil {
		return -1, err
	}
	if adpcmBytes < 0 {
		return -1, fmt.Errorf("adpcm length must be non-negative")
	}

	switch bitsPerSample {
	case BitsPerSample(2):
		return adpcmBytes * 8, nil
	case BitsPerSample(3):
		if adpcmBytes%3 != 0 {
			return -1, fmt.Errorf("adpcm length must be a multiple of 3 bytes for 3 bits/sample decoding")
		}
		return adpcmBytes / 3 * 16, nil
	case BitsPerSample(4):
		return adpcmBytes * 4, nil
	case BitsPerSample(5):
		if adpcmBytes%5 != 0 {
			return -1, fmt.Errorf("adpcm length must be a multiple of 5 bytes for 5 bits/sample decoding")
		}
		return adpcmBytes / 5 * 16, nil
	default:
		return -1, fmt.Errorf("invalid bits per sample: %d", bitsPerSample)
	}
}

func (e *Encoder) Encode(samples []int16) ([]byte, error) {
	pcmBytes := int16sToBytes(samples)
	return e.EncodeBytes(pcmBytes)
}

func (e *Encoder) EncodeBytes(pcmLE []byte) ([]byte, error) {
	if len(pcmLE)%2 != 0 {
		return nil, fmt.Errorf("pcm length must be even")
	}

	size, err := EncodedSize(e.state.bitsPerSample, len(pcmLE))
	if err != nil {
		return nil, err
	}
	if size == 0 {
		return []byte{}, nil
	}

	pkt := make([]byte, size)
	n, err := e.state.encodeToBytes(pcmLE, pkt)
	if err != nil {
		return nil, err
	}
	return pkt[:n], nil
}

func (d *Decoder) Decode(data []byte) ([]int16, error) {
	pcmBytes, err := d.DecodeBytes(data)
	if err != nil {
		return nil, err
	}
	return bytesToInt16s(pcmBytes), nil
}

func (d *Decoder) DecodePostProcessed(data []byte) ([]int16, error) {
	samples, err := d.Decode(data)
	if err != nil {
		return nil, err
	}
	return Postprocess(d.state.bitsPerSample, samples), nil
}

func (d *Decoder) DecodeBytes(data []byte) ([]byte, error) {
	size, err := DecodedSize(d.state.bitsPerSample, len(data))
	if err != nil {
		return nil, err
	}
	if size == 0 {
		return []byte{}, nil
	}

	pcm := make([]byte, size)
	n, err := d.state.decodeToBytes(data, pcm)
	if err != nil {
		return nil, err
	}
	return pcm[:n], nil
}

func (d *Decoder) DecodeBytesPostProcessed(data []byte) ([]byte, error) {
	pcm, err := d.DecodeBytes(data)
	if err != nil {
		return nil, err
	}
	return PostprocessBytes(d.state.bitsPerSample, pcm)
}

func (state_ptr *codecState) encodeToBytes(pcm []byte, pkt []byte) (int, error) {
	length := len(pcm)
	size, err := EncodedSize(state_ptr.bitsPerSample, length)
	if err != nil {
		return -1, err
	}
	if size == 0 {
		return 0, nil
	}

	var (
		n    int
		bits uint
	)
	switch state_ptr.bitsPerSample {
	case BitsPerSample(2):
		bits = 2
		for i := 0; i < length; i += 8 {
			samples := [4]uint32{
				uint32(state_ptr.encodeBits2(int(int16(binary.LittleEndian.Uint16(pcm[i:]))))),
				uint32(state_ptr.encodeBits2(int(int16(binary.LittleEndian.Uint16(pcm[i+2:]))))),
				uint32(state_ptr.encodeBits2(int(int16(binary.LittleEndian.Uint16(pcm[i+4:]))))),
				uint32(state_ptr.encodeBits2(int(int16(binary.LittleEndian.Uint16(pcm[i+6:]))))),
			}
			n += packCodewordsLE(pkt[n:], samples[:], bits)
		}
	case BitsPerSample(3):
		bits = 3
		for i := 0; i < length; i += 16 {
			samples := [8]uint32{
				uint32(state_ptr.encodeBits3(int(int16(binary.LittleEndian.Uint16(pcm[i:]))))),
				uint32(state_ptr.encodeBits3(int(int16(binary.LittleEndian.Uint16(pcm[i+2:]))))),
				uint32(state_ptr.encodeBits3(int(int16(binary.LittleEndian.Uint16(pcm[i+4:]))))),
				uint32(state_ptr.encodeBits3(int(int16(binary.LittleEndian.Uint16(pcm[i+6:]))))),
				uint32(state_ptr.encodeBits3(int(int16(binary.LittleEndian.Uint16(pcm[i+8:]))))),
				uint32(state_ptr.encodeBits3(int(int16(binary.LittleEndian.Uint16(pcm[i+10:]))))),
				uint32(state_ptr.encodeBits3(int(int16(binary.LittleEndian.Uint16(pcm[i+12:]))))),
				uint32(state_ptr.encodeBits3(int(int16(binary.LittleEndian.Uint16(pcm[i+14:]))))),
			}
			n += packCodewordsLE(pkt[n:], samples[:], bits)
		}
	case BitsPerSample(4):
		bits = 4
		for i := 0; i < length; i += 4 {
			samples := [2]uint32{
				uint32(state_ptr.encodeBits4(int(int16(binary.LittleEndian.Uint16(pcm[i:]))))),
				uint32(state_ptr.encodeBits4(int(int16(binary.LittleEndian.Uint16(pcm[i+2:]))))),
			}
			n += packCodewordsLE(pkt[n:], samples[:], bits)
		}
	case BitsPerSample(5):
		bits = 5
		for i := 0; i < length; i += 16 {
			samples := [8]uint32{
				uint32(state_ptr.encodeBits5(int(int16(binary.LittleEndian.Uint16(pcm[i:]))))),
				uint32(state_ptr.encodeBits5(int(int16(binary.LittleEndian.Uint16(pcm[i+2:]))))),
				uint32(state_ptr.encodeBits5(int(int16(binary.LittleEndian.Uint16(pcm[i+4:]))))),
				uint32(state_ptr.encodeBits5(int(int16(binary.LittleEndian.Uint16(pcm[i+6:]))))),
				uint32(state_ptr.encodeBits5(int(int16(binary.LittleEndian.Uint16(pcm[i+8:]))))),
				uint32(state_ptr.encodeBits5(int(int16(binary.LittleEndian.Uint16(pcm[i+10:]))))),
				uint32(state_ptr.encodeBits5(int(int16(binary.LittleEndian.Uint16(pcm[i+12:]))))),
				uint32(state_ptr.encodeBits5(int(int16(binary.LittleEndian.Uint16(pcm[i+14:]))))),
			}
			n += packCodewordsLE(pkt[n:], samples[:], bits)
		}
	default:
		return -1, fmt.Errorf("invalid bits per sample: %d", state_ptr.bitsPerSample)
	}

	return n, nil
}

func (state_ptr *codecState) decodeToBytes(bitstream []byte, pcm []byte) (int, error) {
	inputLen := len(bitstream)
	pcmSize, err := DecodedSize(state_ptr.bitsPerSample, inputLen)
	if err != nil {
		return -1, err
	}
	if pcmSize == 0 {
		return 0, nil
	}

	var bits uint
	switch state_ptr.bitsPerSample {
	case BitsPerSample(2):
		bits = 2
		for i := 0; i < inputLen; i++ {
			var samples [4]uint32
			unpackCodewordsLE(bitstream[i:i+1], samples[:], bits)
			binary.LittleEndian.PutUint16(pcm[i*8:], uint16(state_ptr.decodeBits2(int(samples[0]))))
			binary.LittleEndian.PutUint16(pcm[i*8+2:], uint16(state_ptr.decodeBits2(int(samples[1]))))
			binary.LittleEndian.PutUint16(pcm[i*8+4:], uint16(state_ptr.decodeBits2(int(samples[2]))))
			binary.LittleEndian.PutUint16(pcm[i*8+6:], uint16(state_ptr.decodeBits2(int(samples[3]))))
		}
	case BitsPerSample(3):
		bits = 3
		for i := 0; i < inputLen; i += 3 {
			var samples [8]uint32
			unpackCodewordsLE(bitstream[i:i+3], samples[:], bits)
			n := i / 3 * 16
			binary.LittleEndian.PutUint16(pcm[n:], uint16(state_ptr.decodeBits3(int(samples[0]))))
			binary.LittleEndian.PutUint16(pcm[n+2:], uint16(state_ptr.decodeBits3(int(samples[1]))))
			binary.LittleEndian.PutUint16(pcm[n+4:], uint16(state_ptr.decodeBits3(int(samples[2]))))
			binary.LittleEndian.PutUint16(pcm[n+6:], uint16(state_ptr.decodeBits3(int(samples[3]))))
			binary.LittleEndian.PutUint16(pcm[n+8:], uint16(state_ptr.decodeBits3(int(samples[4]))))
			binary.LittleEndian.PutUint16(pcm[n+10:], uint16(state_ptr.decodeBits3(int(samples[5]))))
			binary.LittleEndian.PutUint16(pcm[n+12:], uint16(state_ptr.decodeBits3(int(samples[6]))))
			binary.LittleEndian.PutUint16(pcm[n+14:], uint16(state_ptr.decodeBits3(int(samples[7]))))
		}
	case BitsPerSample(4):
		bits = 4
		for i := 0; i < inputLen; i++ {
			var samples [2]uint32
			unpackCodewordsLE(bitstream[i:i+1], samples[:], bits)
			binary.LittleEndian.PutUint16(pcm[i*4:], uint16(state_ptr.decodeBits4(int(samples[0]))))
			binary.LittleEndian.PutUint16(pcm[i*4+2:], uint16(state_ptr.decodeBits4(int(samples[1]))))
		}
	case BitsPerSample(5):
		bits = 5
		for i := 0; i < inputLen; i += 5 {
			var samples [8]uint32
			unpackCodewordsLE(bitstream[i:i+5], samples[:], bits)
			n := i / 5 * 16
			binary.LittleEndian.PutUint16(pcm[n:], uint16(state_ptr.decodeBits5(int(samples[0]))))
			binary.LittleEndian.PutUint16(pcm[n+2:], uint16(state_ptr.decodeBits5(int(samples[1]))))
			binary.LittleEndian.PutUint16(pcm[n+4:], uint16(state_ptr.decodeBits5(int(samples[2]))))
			binary.LittleEndian.PutUint16(pcm[n+6:], uint16(state_ptr.decodeBits5(int(samples[3]))))
			binary.LittleEndian.PutUint16(pcm[n+8:], uint16(state_ptr.decodeBits5(int(samples[4]))))
			binary.LittleEndian.PutUint16(pcm[n+10:], uint16(state_ptr.decodeBits5(int(samples[5]))))
			binary.LittleEndian.PutUint16(pcm[n+12:], uint16(state_ptr.decodeBits5(int(samples[6]))))
			binary.LittleEndian.PutUint16(pcm[n+14:], uint16(state_ptr.decodeBits5(int(samples[7]))))
		}
	default:
		return -1, fmt.Errorf("invalid bits per sample: %d", state_ptr.bitsPerSample)
	}

	return pcmSize, nil
}

func validateBitsPerSample(bitsPerSample BitsPerSample) error {
	switch bitsPerSample {
	case BitsPerSample(2), BitsPerSample(3), BitsPerSample(4), BitsPerSample(5):
		return nil
	default:
		return fmt.Errorf("bits per sample must be one of 2, 3, 4, or 5")
	}
}

func int16sToBytes(samples []int16) []byte {
	if len(samples) == 0 {
		return []byte{}
	}

	out := make([]byte, len(samples)*2)
	for i, sample := range samples {
		binary.LittleEndian.PutUint16(out[i*2:], uint16(sample))
	}
	return out
}

func bytesToInt16s(pcm []byte) []int16 {
	if len(pcm) == 0 {
		return []int16{}
	}

	out := make([]int16, len(pcm)/2)
	for i := range out {
		out[i] = int16(binary.LittleEndian.Uint16(pcm[i*2:]))
	}
	return out
}

func packCodewordsLE(dst []byte, codes []uint32, bits uint) int {
	var bitOffset uint
	mask := uint32((1 << bits) - 1)

	for _, code := range codes {
		value := code & mask
		byteIndex := int(bitOffset / 8)
		shift := bitOffset % 8

		dst[byteIndex] |= byte(value << shift)
		if shift+bits > 8 {
			dst[byteIndex+1] |= byte(value >> (8 - shift))
		}

		bitOffset += bits
	}

	return int(bitOffset / 8)
}

func unpackCodewordsLE(src []byte, codes []uint32, bits uint) {
	var bitOffset uint
	mask := uint32((1 << bits) - 1)

	for i := range codes {
		byteIndex := int(bitOffset / 8)
		shift := bitOffset % 8

		value := uint32(src[byteIndex]) >> shift
		if shift+bits > 8 {
			value |= uint32(src[byteIndex+1]) << (8 - shift)
		}

		codes[i] = value & mask
		bitOffset += bits
	}
}
