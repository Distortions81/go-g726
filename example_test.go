package g726_test

import (
	"fmt"

	"github.com/lkmio/g726"
)

func ExampleEncodeBytes() {
	pcm := make([]byte, 16)

	encoder, _ := g726.NewEncoder(4)
	adpcm, _ := encoder.EncodeBytes(pcm)

	decoder, _ := g726.NewDecoder(4)
	out, _ := decoder.DecodeBytes(adpcm)

	fmt.Println(len(adpcm), len(out))
	// Output: 4 16
}
