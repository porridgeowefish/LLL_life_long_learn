package idgen

import (
	"crypto/rand"
	"encoding/binary"
	"strings"
	"time"
)

const alphabet = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

// New returns a type-prefixed ULID compatible identifier without adding a
// third-party dependency. The 128-bit payload is 48 bits of Unix milliseconds
// followed by 80 cryptographically random bits, encoded with Crockford base32.
func New(prefix string) string {
	var raw [16]byte
	ms := uint64(time.Now().UTC().UnixMilli())
	raw[0] = byte(ms >> 40)
	raw[1] = byte(ms >> 32)
	raw[2] = byte(ms >> 24)
	raw[3] = byte(ms >> 16)
	raw[4] = byte(ms >> 8)
	raw[5] = byte(ms)
	if _, err := rand.Read(raw[6:]); err != nil {
		binary.BigEndian.PutUint64(raw[8:], uint64(time.Now().UnixNano()))
	}
	return strings.TrimSuffix(prefix, "_") + "_" + encode(raw)
}

func encode(raw [16]byte) string {
	// ULID is 26 base32 chars (130 bits); the leading two bits are zero.
	out := make([]byte, 26)
	var buffer uint32
	bits := 2
	idx := 0
	for _, b := range raw {
		buffer = (buffer << 8) | uint32(b)
		bits += 8
		for bits >= 5 && idx < len(out) {
			bits -= 5
			out[idx] = alphabet[(buffer>>bits)&31]
			idx++
			if bits == 0 {
				buffer = 0
			} else {
				buffer &= (1 << bits) - 1
			}
		}
	}
	for idx < len(out) {
		out[idx] = alphabet[0]
		idx++
	}
	return string(out)
}
