# g726

Forked from [github.com/lkmio/g726](https://github.com/lkmio/g726).

This package provides G.726 ADPCM encode and decode APIs for mono 8 kHz audio with explicit bits-per-sample selection.

Supported code sizes:

- `2` bits/sample
- `3` bits/sample
- `4` bits/sample
- `5` bits/sample

The package exposes:

- stateful `Encoder` and `Decoder` types for streaming use
- one-shot package helpers for simple buffer-to-buffer conversions
- size helpers for preallocating encoded and decoded buffers

## Install

```bash
go get github.com/Distortions81/g726
```

## Overview

G.726 is a stateful ADPCM codec. Each encoded sample updates predictor state, step size, and adaptation state for the next sample.

That has two practical consequences:

- A single `Encoder` or `Decoder` instance should be reused for a continuous stream.
- If you split a stream into chunks, decode or encode them with the same stateful instance in order.

If you want independent packets, create a fresh `Encoder` or `Decoder` for each packet, or call `Reset()` between independent segments.

## Quick Start

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

## API

### Stateful API

Use the stateful API for streaming audio.

```go
encoder, err := g726.NewEncoder(4)
decoder, err := g726.NewDecoder(4)
```

Encode signed PCM samples:

```go
adpcm, err := encoder.Encode(samples)
```

Encode little-endian PCM bytes:

```go
adpcm, err := encoder.EncodeBytes(pcmLE)
```

Decode to signed samples:

```go
samples, err := decoder.Decode(adpcm)
```

Decode to signed samples with optional waveform cleanup:

```go
samples, err := decoder.DecodePostProcessed(adpcm)
```

Decode to little-endian PCM bytes:

```go
pcmLE, err := decoder.DecodeBytes(adpcm)
```

Decode to little-endian PCM bytes with optional waveform cleanup:

```go
pcmLE, err := decoder.DecodeBytesPostProcessed(adpcm)
```

Reset state before starting a new independent stream:

```go
encoder.Reset()
decoder.Reset()
```

### One-shot helpers

Use the package-level helpers when you have a single self-contained buffer and do not need to preserve codec state across calls.

```go
adpcm, err := g726.Encode(bitsPerSample, samples)
adpcm, err := g726.EncodeBytes(bitsPerSample, pcmLE)
samples, err := g726.Decode(bitsPerSample, adpcm)
pcmLE, err := g726.DecodeBytes(bitsPerSample, adpcm)
samples, err := g726.DecodePostProcessed(bitsPerSample, adpcm)
pcmLE, err := g726.DecodeBytesPostProcessed(bitsPerSample, adpcm)
```

### Size helpers

```go
encodedSize, err := g726.EncodedSize(bitsPerSample, len(pcmLE))
decodedSize, err := g726.DecodedSize(bitsPerSample, len(adpcm))
```

`EncodedSize` and `DecodedSize` validate framing rules for the selected bit depth.

## Input Rules

- `bitsPerSample` must be `2`, `3`, `4`, or `5`
- PCM byte input must be little-endian signed 16-bit and have even length
- The codec expects mono sample data
- Encoding alignment:
  - `2` bits/sample: PCM bytes must be a multiple of `8` bytes (`4` samples)
  - `3` bits/sample: PCM bytes must be a multiple of `16` bytes (`8` samples)
  - `4` bits/sample: PCM bytes must be a multiple of `4` bytes (`2` samples)
  - `5` bits/sample: PCM bytes must be a multiple of `16` bytes (`8` samples)
- Decoding alignment:
  - `2` bits/sample: any byte length
  - `3` bits/sample: ADPCM bytes must be a multiple of `3`
  - `4` bits/sample: any byte length
  - `5` bits/sample: ADPCM bytes must be a multiple of `5`

## Streaming Notes

- `Encoder` and `Decoder` are not interchangeable across bit depths.
- Reusing a stateful instance across unrelated clips will leak predictor state between them.
- `Reset()` restores the initial codec state for the configured bit depth.

## State Notes

Recent work in this fork has focused on state handling bugs, especially around:

- preserving adaptive state across streaming calls
- maintaining correct packed-group behavior for non-4-bit modes
- avoiding invalid decoded PCM wrapping on extreme outputs

The codec is still being exercised against loud-signal edge cases, especially for lower bit depths where overload artifacts are easier to trigger.

## Post-Processing

This package now includes an optional decode-side waveform cleanup pass:

- `decoder.DecodePostProcessed(...)`
- `decoder.DecodeBytesPostProcessed(...)`
- `g726.DecodePostProcessed(...)`
- `g726.DecodeBytesPostProcessed(...)`
- `g726.Postprocess(bitsPerSample, samples)`
- `g726.PostprocessBytes(bitsPerSample, pcmLE)`

The post-process is conservative and targets abrupt single-sample spikes and unrealistic jump transitions that can sound like pops. It is intended as an opt-in cleanup stage, not a replacement for the raw codec output.

## Sample Conversion Commands

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
