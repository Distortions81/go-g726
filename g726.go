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

func DecodeBytes(bitsPerSample BitsPerSample, data []byte) ([]byte, error) {
	decoder, err := NewDecoder(bitsPerSample)
	if err != nil {
		return nil, err
	}
	return decoder.DecodeBytes(data)
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

func (state_ptr *codecState) encodeToBytes(pcm []byte, pkt []byte) (int, error) {
	length := len(pcm)
	size, err := EncodedSize(state_ptr.bitsPerSample, length)
	if err != nil {
		return -1, err
	}
	if size == 0 {
		return 0, nil
	}

	var n int
	switch state_ptr.bitsPerSample {
	case BitsPerSample(2):
		for i := 0; i < length; i += 8 {
			a := state_ptr.encodeBits2(int(int16(binary.LittleEndian.Uint16(pcm[i:]))))
			b := state_ptr.encodeBits2(int(int16(binary.LittleEndian.Uint16(pcm[i+2:]))))
			c := state_ptr.encodeBits2(int(int16(binary.LittleEndian.Uint16(pcm[i+4:]))))
			d := state_ptr.encodeBits2(int(int16(binary.LittleEndian.Uint16(pcm[i+6:]))))
			pkt[n] = byte((a << 6) | (b << 4) | (c << 2) | d)
			n++
		}
	case BitsPerSample(3):
		for i := 0; i < length; i += 16 {
			s0 := state_ptr.encodeBits3(int(int16(binary.LittleEndian.Uint16(pcm[i:]))))
			s1 := state_ptr.encodeBits3(int(int16(binary.LittleEndian.Uint16(pcm[i+2:]))))
			s2 := state_ptr.encodeBits3(int(int16(binary.LittleEndian.Uint16(pcm[i+4:]))))
			s3 := state_ptr.encodeBits3(int(int16(binary.LittleEndian.Uint16(pcm[i+6:]))))
			s4 := state_ptr.encodeBits3(int(int16(binary.LittleEndian.Uint16(pcm[i+8:]))))
			s5 := state_ptr.encodeBits3(int(int16(binary.LittleEndian.Uint16(pcm[i+10:]))))
			s6 := state_ptr.encodeBits3(int(int16(binary.LittleEndian.Uint16(pcm[i+12:]))))
			s7 := state_ptr.encodeBits3(int(int16(binary.LittleEndian.Uint16(pcm[i+14:]))))

			pkt[n] = byte(s0<<5) | byte(s1<<2) | byte(s2>>1)
			pkt[n+1] = byte((s2&1)<<7) | byte(s3<<4) | byte(s4<<1) | byte(s5>>2)
			pkt[n+2] = byte((s5&3)<<6) | byte(s6<<3) | byte(s7)
			n += 3
		}
	case BitsPerSample(4):
		for i := 0; i < length; i += 4 {
			a := state_ptr.encodeBits4(int(int16(binary.LittleEndian.Uint16(pcm[i:]))))
			b := state_ptr.encodeBits4(int(int16(binary.LittleEndian.Uint16(pcm[i+2:]))))
			pkt[n] = byte((a << 4) | b)
			n++
		}
	case BitsPerSample(5):
		for i := 0; i < length; i += 16 {
			s0 := state_ptr.encodeBits5(int(int16(binary.LittleEndian.Uint16(pcm[i:]))))
			s1 := state_ptr.encodeBits5(int(int16(binary.LittleEndian.Uint16(pcm[i+2:]))))
			s2 := state_ptr.encodeBits5(int(int16(binary.LittleEndian.Uint16(pcm[i+4:]))))
			s3 := state_ptr.encodeBits5(int(int16(binary.LittleEndian.Uint16(pcm[i+6:]))))
			s4 := state_ptr.encodeBits5(int(int16(binary.LittleEndian.Uint16(pcm[i+8:]))))
			s5 := state_ptr.encodeBits5(int(int16(binary.LittleEndian.Uint16(pcm[i+10:]))))
			s6 := state_ptr.encodeBits5(int(int16(binary.LittleEndian.Uint16(pcm[i+12:]))))
			s7 := state_ptr.encodeBits5(int(int16(binary.LittleEndian.Uint16(pcm[i+14:]))))

			pkt[n] = byte((s0 << 3) | (s1 >> 2))
			pkt[n+1] = byte(((s1 & 0x03) << 6) | (s2 << 1) | (s3 >> 4))
			pkt[n+2] = byte(((s3 & 0x0F) << 4) | (s4 >> 1))
			pkt[n+3] = byte(((s4 & 0x01) << 7) | (s5 << 2) | (s6 >> 3))
			pkt[n+4] = byte(((s6 & 0x07) << 5) | s7)
			n += 5
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

	switch state_ptr.bitsPerSample {
	case BitsPerSample(2):
		for i := 0; i < inputLen; i++ {
			a := (bitstream[i] & byte(192)) >> 6
			b := (bitstream[i] & byte(48)) >> 4
			c := (bitstream[i] & byte(12)) >> 2
			dv := bitstream[i] & byte(3)

			binary.LittleEndian.PutUint16(pcm[i*8:], uint16(state_ptr.decodeBits2(int(a))))
			binary.LittleEndian.PutUint16(pcm[i*8+2:], uint16(state_ptr.decodeBits2(int(b))))
			binary.LittleEndian.PutUint16(pcm[i*8+4:], uint16(state_ptr.decodeBits2(int(c))))
			binary.LittleEndian.PutUint16(pcm[i*8+6:], uint16(state_ptr.decodeBits2(int(dv))))
		}
	case BitsPerSample(3):
		for i := 0; i < inputLen; i += 3 {
			b0 := bitstream[i]
			b1 := bitstream[i+1]
			b2 := bitstream[i+2]

			s0 := (b0 & 0xE0) >> 5
			s1 := (b0 & 0x1C) >> 2
			s2 := ((b0 & 0x03) << 1) | ((b1 & 0x80) >> 7)
			s3 := (b1 & 0x70) >> 4
			s4 := (b1 & 0x0E) >> 1
			s5 := ((b1 & 0x01) << 2) | ((b2 & 0xC0) >> 6)
			s6 := (b2 & 0x38) >> 3
			s7 := b2 & 0x07

			n := i / 3 * 16
			binary.LittleEndian.PutUint16(pcm[n:], uint16(state_ptr.decodeBits3(int(s0))))
			binary.LittleEndian.PutUint16(pcm[n+2:], uint16(state_ptr.decodeBits3(int(s1))))
			binary.LittleEndian.PutUint16(pcm[n+4:], uint16(state_ptr.decodeBits3(int(s2))))
			binary.LittleEndian.PutUint16(pcm[n+6:], uint16(state_ptr.decodeBits3(int(s3))))
			binary.LittleEndian.PutUint16(pcm[n+8:], uint16(state_ptr.decodeBits3(int(s4))))
			binary.LittleEndian.PutUint16(pcm[n+10:], uint16(state_ptr.decodeBits3(int(s5))))
			binary.LittleEndian.PutUint16(pcm[n+12:], uint16(state_ptr.decodeBits3(int(s6))))
			binary.LittleEndian.PutUint16(pcm[n+14:], uint16(state_ptr.decodeBits3(int(s7))))
		}
	case BitsPerSample(4):
		for i := 0; i < inputLen; i++ {
			a := (bitstream[i] & byte(240)) >> 4
			b := bitstream[i] & byte(15)
			binary.LittleEndian.PutUint16(pcm[i*4:], uint16(state_ptr.decodeBits4(int(a))))
			binary.LittleEndian.PutUint16(pcm[i*4+2:], uint16(state_ptr.decodeBits4(int(b))))
		}
	case BitsPerSample(5):
		for i := 0; i < inputLen; i += 5 {
			b0 := bitstream[i]
			b1 := bitstream[i+1]
			b2 := bitstream[i+2]
			b3 := bitstream[i+3]
			b4 := bitstream[i+4]

			s0 := (b0 & 0xF8) >> 3
			s1 := ((b0 & 0x07) << 2) | ((b1 & 0xC0) >> 6)
			s2 := (b1 & 0x3E) >> 1
			s3 := ((b1 & 0x01) << 4) | ((b2 & 0xF0) >> 4)
			s4 := ((b2 & 0x0F) << 1) | ((b3 & 0x80) >> 7)
			s5 := (b3 & 0x7C) >> 2
			s6 := ((b3 & 0x03) << 3) | ((b4 & 0xE0) >> 5)
			s7 := b4 & 0x1F

			n := i / 5 * 16
			binary.LittleEndian.PutUint16(pcm[n:], uint16(state_ptr.decodeBits5(int(s0))))
			binary.LittleEndian.PutUint16(pcm[n+2:], uint16(state_ptr.decodeBits5(int(s1))))
			binary.LittleEndian.PutUint16(pcm[n+4:], uint16(state_ptr.decodeBits5(int(s2))))
			binary.LittleEndian.PutUint16(pcm[n+6:], uint16(state_ptr.decodeBits5(int(s3))))
			binary.LittleEndian.PutUint16(pcm[n+8:], uint16(state_ptr.decodeBits5(int(s4))))
			binary.LittleEndian.PutUint16(pcm[n+10:], uint16(state_ptr.decodeBits5(int(s5))))
			binary.LittleEndian.PutUint16(pcm[n+12:], uint16(state_ptr.decodeBits5(int(s6))))
			binary.LittleEndian.PutUint16(pcm[n+14:], uint16(state_ptr.decodeBits5(int(s7))))
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
