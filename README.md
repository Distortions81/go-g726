# g726

Forked from [github.com/lkmio/g726](https://github.com/lkmio/g726).

Simple G.726 ADPCM encode/decode APIs with mandatory bits-per-sample selection.

Supported code sizes:

- `2` bits/sample
- `3` bits/sample
- `4` bits/sample
- `5` bits/sample

The package exposes stateful `Encoder` and `Decoder` types for streaming, plus one-shot helper functions for convenience.

## Install

```bash
go get github.com/Distortions81/g726
```

## Quick start

```go
package main

import (
	"fmt"
	"os"

	"github.com/Distortions81/g726"
)

func main() {
	pcm, err := os.ReadFile("audio-samples.pcm")
	if err != nil {
		panic(err)
	}

	encoder, err := g726.NewEncoder(4)
	if err != nil {
		panic(err)
	}

	adpcm, err := encoder.EncodeBytes(pcm)
	if err != nil {
		panic(err)
	}

	decoder, err := g726.NewDecoder(4)
	if err != nil {
		panic(err)
	}

	outPCM, err := decoder.DecodeBytes(adpcm)
	if err != nil {
		panic(err)
	}

	fmt.Println(len(outPCM))
}
```

## API summary

```go
encoder, err := g726.NewEncoder(bitsPerSample)
decoder, err := g726.NewDecoder(bitsPerSample)

adpcm, err := encoder.Encode(samples)
adpcm, err := encoder.EncodeBytes(pcmLE)

samples, err := decoder.Decode(adpcm)
pcmLE, err := decoder.DecodeBytes(adpcm)

encoder.Reset()
decoder.Reset()

adpcm, err := g726.Encode(bitsPerSample, samples)
adpcm, err := g726.EncodeBytes(bitsPerSample, pcmLE)
samples, err := g726.Decode(bitsPerSample, adpcm)
pcmLE, err := g726.DecodeBytes(bitsPerSample, adpcm)

encodedSize, err := g726.EncodedSize(bitsPerSample, len(pcmLE))
decodedSize, err := g726.DecodedSize(bitsPerSample, len(adpcm))
```

## Input rules

- `bitsPerSample` must be `2`, `3`, `4`, or `5`
- PCM byte input must be little-endian signed 16-bit and have even length
- Encoding alignment:
  - `2` bits/sample: PCM bytes must be a multiple of `8`
  - `3` bits/sample: PCM bytes must be a multiple of `16`
  - `4` bits/sample: PCM bytes must be a multiple of `4`
  - `5` bits/sample: PCM bytes must be a multiple of `16`
- Decoding alignment:
  - `2` bits/sample: any byte length
  - `3` bits/sample: ADPCM bytes must be a multiple of `3`
  - `4` bits/sample: any byte length
  - `5` bits/sample: ADPCM bytes must be a multiple of `5`

## Sample conversion commands

Create a mono 8 kHz PCM fixture:

```bash
ffmpeg -i input.mp3 -ar 8000 -ac 1 -acodec pcm_s16le -f s16le audio-samples.pcm
```

Play raw PCM:

```bash
ffplay -ar 8000 -ac 1 -f s16le -i audio-samples.pcm
```

Play G.726 produced with `4` bits/sample:

```bash
ffplay -f g726le -ar 8000 -ac 1 -code_size 4 -i audio-samples.g726
```
