package crypt

import (
	"bytes"
	"crypto/md5" //nolint:gosec // ISO 32000-1 §7.6.3 mandates MD5
)

// newRC4orAES handles the standard handler revisions 2-4 (RC4 and, via V4
// crypt filters, AES-128). It authenticates the supplied user password, then
// the supplied owner password, deriving the file key with Algorithm 2.
func newRC4orAES(c Config, password []byte) (*Decryptor, error) {
	keyLen := 5
	if c.R >= 3 && c.Length >= 40 {
		keyLen = c.Length / 8
	}
	if keyLen < 5 || keyLen > 16 {
		keyLen = 5
	}

	// Try the supplied user password (Algorithm 2).
	key := fileKeyRC4(password, c, keyLen)
	if !userKeyValid(key, c) {
		// Fall back to the supplied owner password (Algorithm 7): recover the
		// user password from /O, then re-derive and re-check.
		if userPw, ok := ownerRecoverUserPw(password, c, keyLen); ok {
			key = fileKeyRC4(userPw, c, keyLen)
			if !userKeyValid(key, c) {
				return nil, ErrPasswordRequired
			}
		} else {
			return nil, ErrPasswordRequired
		}
	}

	stm, str := RC4, RC4
	if c.V == 4 {
		stm = methodForFilter(c, c.StmF)
		str = methodForFilter(c, c.StrF)
	}
	return &Decryptor{key: key, stmF: stm, strF: str}, nil
}

// fileKeyRC4 implements Algorithm 2: MD5 of the padded password, /O, /P
// (little-endian), /ID[0], and — for R≥4 without metadata encryption — an
// extra 0xFFFFFFFF, then (R≥3) 50 more MD5 rounds over the first keyLen
// bytes.
func fileKeyRC4(pw []byte, c Config, keyLen int) []byte {
	h := md5.New() //nolint:gosec // ISO 32000-1 §7.6.3 Algorithm 2
	h.Write(padPassword(pw))
	h.Write(c.O)
	p := uint32(c.P) //nolint:gosec // /P is a 32-bit flag field
	h.Write([]byte{byte(p & 0xFF), byte(p >> 8 & 0xFF), byte(p >> 16 & 0xFF), byte(p >> 24 & 0xFF)})
	h.Write(c.ID)
	if c.R >= 4 && !c.EncryptMeta {
		h.Write([]byte{0xFF, 0xFF, 0xFF, 0xFF})
	}
	key := h.Sum(nil)[:keyLen]
	if c.R >= 3 {
		for range 50 {
			key = hashN(key, keyLen)
		}
	}
	return key
}

// userKeyValid checks the derived key against /U (Algorithm 4 for R2,
// Algorithm 5 for R3/4).
func userKeyValid(key []byte, c Config) bool {
	if c.R == 2 {
		want := rc4Crypt(key, passwordPad)
		return bytes.HasPrefix(c.U, want) || bytes.Equal(want, c.U)
	}
	h := md5.New() //nolint:gosec // ISO 32000-1 §7.6.3 Algorithm 5
	h.Write(passwordPad)
	h.Write(c.ID)
	digest := rc4Crypt(key, h.Sum(nil))
	for i := 1; i <= 19; i++ {
		digest = rc4Crypt(xorKey(key, byte(i)), digest)
	}
	// Only the first 16 bytes of /U are the checksum; the rest is padding.
	return len(c.U) >= 16 && bytes.Equal(digest[:16], c.U[:16])
}

// ownerRecoverUserPw implements Algorithm 7: derive the RC4 key from the
// supplied owner password and use it to decrypt /O back to the user password.
func ownerRecoverUserPw(ownerPw []byte, c Config, keyLen int) ([]byte, bool) {
	if len(c.O) < 32 {
		return nil, false
	}
	key := hashN(padPassword(ownerPw), 16)
	if c.R >= 3 {
		for range 50 {
			key = hashN(key, 16)
		}
	}
	key = key[:keyLen]
	userPw := make([]byte, 32)
	copy(userPw, c.O[:32])
	if c.R == 2 {
		userPw = rc4Crypt(key, userPw)
	} else {
		for i := 19; i >= 0; i-- {
			userPw = rc4Crypt(xorKey(key, byte(i)), userPw)
		}
	}
	return userPw, true
}

// methodForFilter resolves a /StmF or /StrF crypt-filter name to a Method
// via the /CF dictionary.
func methodForFilter(c Config, name string) Method {
	switch name {
	case "", "Identity":
		return Identity
	}
	cf, ok := c.CF[name]
	if !ok {
		return Identity
	}
	switch cf.CFM {
	case "V2":
		return RC4
	case "AESV2":
		return AESV2
	case "AESV3":
		return AESV3
	default:
		return Identity
	}
}

// padPassword truncates or pads a password to 32 bytes with the standard
// padding string (Algorithm 2, step a).
func padPassword(pw []byte) []byte {
	out := make([]byte, 32)
	n := copy(out, pw)
	copy(out[n:], passwordPad)
	return out
}

func xorKey(key []byte, x byte) []byte {
	out := make([]byte, len(key))
	for i, b := range key {
		out[i] = b ^ x
	}
	return out
}
