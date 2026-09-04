package mutation

import (
	"bytes"
	"encoding/binary"
	"math/rand"
)

func bitFlip(data []byte, rng *rand.Rand) []byte {
	out := append([]byte(nil), data...)
	i := rng.Intn(len(out))
	bit := uint8(1 << uint(rng.Intn(8)))
	out[i] ^= bit
	return out
}

func byteFlip(data []byte, rng *rand.Rand) []byte {
	out := append([]byte(nil), data...)
	i := rng.Intn(len(out))
	out[i] = byte(rng.Intn(256))
	return out
}

func arithmetic(data []byte, rng *rand.Rand) []byte {
	out := append([]byte(nil), data...)
	i := rng.Intn(len(out))
	delta := int(rng.Intn(35)) - 17
	if delta == 0 {
		delta = 1
	}
	out[i] = byte(int(out[i]) + delta)
	return out
}

func incDec(data []byte, rng *rand.Rand) []byte {
	out := append([]byte(nil), data...)
	i := rng.Intn(len(out))
	if rng.Intn(2) == 0 {
		out[i]++
	} else {
		out[i]--
	}
	return out
}

func interestingValue(data []byte, rng *rand.Rand) []byte {
	out := append([]byte(nil), data...)
	want := interestingInts[rng.Intn(len(interestingInts))]
	// Choose a write width aligned to available bytes.
	width := 1 + rng.Intn(minInt(8, len(out)))
	start := rng.Intn(len(out) - width + 1)
	buf := make([]byte, 8)
	if rng.Intn(2) == 0 {
		binary.LittleEndian.PutUint64(buf, want)
	} else {
		binary.BigEndian.PutUint64(buf, want)
	}
	copy(out[start:start+width], buf[8-width:])
	return out
}

func boundary(data []byte, rng *rand.Rand) []byte {
	out := append([]byte(nil), data...)
	vals := []byte{0x00, 0x01, 0x7f, 0x80, 0xfe, 0xff}
	i := rng.Intn(len(out))
	out[i] = vals[rng.Intn(len(vals))]
	return out
}

func blockInsert(data []byte, rng *rand.Rand) []byte {
	block := data[rng.Intn(len(data))]
	length := 1 + rng.Intn(8)
	rep := make([]byte, length)
	for i := range rep {
		rep[i] = block
	}
	at := rng.Intn(len(data) + 1)
	out := make([]byte, 0, len(data)+length)
	out = append(out, data[:at]...)
	out = append(out, rep...)
	out = append(out, data[at:]...)
	return out
}

func blockDelete(data []byte, rng *rand.Rand) []byte {
	if len(data) <= 1 {
		return data
	}
	length := 1 + rng.Intn(minInt(len(data), 16))
	at := rng.Intn(len(data) - length + 1)
	out := make([]byte, 0, len(data)-length)
	out = append(out, data[:at]...)
	out = append(out, data[at+length:]...)
	return out
}

func blockDuplicate(data []byte, rng *rand.Rand) []byte {
	length := 1 + rng.Intn(minInt(len(data), 16))
	at := rng.Intn(len(data) - length + 1)
	block := append([]byte(nil), data[at:at+length]...)
	insertAt := rng.Intn(len(data) + 1)
	out := make([]byte, 0, len(data)+length)
	out = append(out, data[:insertAt]...)
	out = append(out, block...)
	out = append(out, data[insertAt:]...)
	return out
}

func blockReplace(data []byte, rng *rand.Rand) []byte {
	length := 1 + rng.Intn(minInt(len(data), 16))
	at := rng.Intn(len(data) - length + 1)
	block := data[rng.Intn(len(data))]
	out := append([]byte(nil), data...)
	for i := at; i < at+length && i < len(out); i++ {
		out[i] = block
	}
	return out
}

func byteInsert(data []byte, rng *rand.Rand) []byte {
	at := rng.Intn(len(data) + 1)
	b := byte(rng.Intn(256))
	out := make([]byte, 0, len(data)+1)
	out = append(out, data[:at]...)
	out = append(out, b)
	out = append(out, data[at:]...)
	return out
}

func byteDelete(data []byte, rng *rand.Rand) []byte {
	if len(data) <= 1 {
		return data
	}
	at := rng.Intn(len(data))
	out := make([]byte, 0, len(data)-1)
	out = append(out, data[:at]...)
	out = append(out, data[at+1:]...)
	return out
}

func chunkShuffle(data []byte, rng *rand.Rand) []byte {
	// Split into 2-6 chunks and reorder them, preserving content atoms.
	n := 2 + rng.Intn(5)
	if n > len(data) {
		n = len(data)
	}
	if n < 2 {
		return data
	}
	sizes := make([]int, n)
	per := len(data) / n
	rem := len(data) % n
	idx := 0
	for i := 0; i < n; i++ {
		sizes[i] = per
		if i < rem {
			sizes[i]++
		}
		idx += sizes[i]
	}
	chunks := make([][]byte, n)
	cut := 0
	for i := 0; i < n; i++ {
		chunks[i] = append([]byte(nil), data[cut:cut+sizes[i]]...)
		cut += sizes[i]
	}
	rng.Shuffle(n, func(i, j int) { chunks[i], chunks[j] = chunks[j], chunks[i] })
	var out []byte
	for _, c := range chunks {
		out = append(out, c...)
	}
	return out
}

func splice(data []byte, _ *Engine) []byte {
	// Splice is a two-source recombination; with a single source we keep the
	// same bytes (documented limitation) so the op is still safe.
	return append([]byte(nil), data...)
}

func lengthMutation(data []byte, rng *rand.Rand) []byte {
	// Trim or extend to a boundary-ish length; used to probe size-sensitive
	// parsers.
	if len(data) == 0 {
		return data
	}
	switch rng.Intn(3) {
	case 0:
		if len(data) > 1 {
			return data[:1]
		}
	case 1:
		if len(data) > 4 {
			return data[:4]
		}
	default:
		return data
	}
	return data
}

var delimiters = []byte{'\n', '\r', '\t', ' ', 0, ';', ',', ':', '/', '\\', '"', '\''}

func delimiterMutation(data []byte, rng *rand.Rand) []byte {
	out := append([]byte(nil), data...)
	at := rng.Intn(len(out))
	d := delimiters[rng.Intn(len(delimiters))]
	// Toggle a delimiter, or insert one.
	if rng.Intn(2) == 0 && at < len(out) {
		out[at] = d
	} else {
		out = append(out[:at], append([]byte{d}, out[at:]...)...)
	}
	return out
}

func dictionaryInsert(data []byte, e *Engine) []byte {
	if len(e.dictionary) == 0 {
		return append([]byte{0x41}, data...) // 'A' fallback
	}
	tok := e.dictionary[e.rng.Intn(len(e.dictionary))]
	at := e.rng.Intn(len(data) + 1)
	out := make([]byte, 0, len(data)+len(tok))
	out = append(out, data[:at]...)
	out = append(out, tok...)
	out = append(out, data[at:]...)
	return out
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

var _ = bytes.Compare
