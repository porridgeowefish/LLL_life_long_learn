package crypt

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"crypto/sha512"
	"hash"
)

// newAES256 handles the AES-256 standard handler (revisions 5 and 6, ISO
// 32000-2 §7.6.4.3.4). It authenticates the supplied user password, then the
// supplied owner password, and unwraps the file key from /UE or /OE.
func newAES256(c Config, password []byte) (*Decryptor, error) {
	if len(c.U) < 48 || len(c.O) < 48 {
		return nil, ErrPasswordRequired
	}
	// Revisions 5 and 6 use the UTF-8 password bytes, truncated to 127 bytes.
	if len(password) > 127 {
		password = password[:127]
	}
	uHash, uValSalt, uKeySalt := c.U[:32], c.U[32:40], c.U[40:48]

	// User password.
	if bytes.Equal(hash2B(password, uValSalt, nil, c.R), uHash) {
		ik := hash2B(password, uKeySalt, nil, c.R)
		if key := aesNoPadCBC(ik, c.UE); key != nil {
			return aes256Decryptor(c, key), nil
		}
	}

	// Owner password: its salts hash together with the full 48-byte /U.
	oHash, oValSalt, oKeySalt := c.O[:32], c.O[32:40], c.O[40:48]
	if bytes.Equal(hash2B(password, oValSalt, c.U[:48], c.R), oHash) {
		ik := hash2B(password, oKeySalt, c.U[:48], c.R)
		if key := aesNoPadCBC(ik, c.OE); key != nil {
			return aes256Decryptor(c, key), nil
		}
	}
	return nil, ErrPasswordRequired
}

// aes256Decryptor selects the string and stream methods through the
// crypt filters, as for V4. A revision 5 or 6 file names its filters; one
// that omits them is read as AES-256 throughout, as before crypt filters
// were consulted.
func aes256Decryptor(c Config, key []byte) *Decryptor {
	method := func(name string) Method {
		if name == "" {
			return AESV3
		}
		return methodForFilter(c, name)
	}
	return &Decryptor{key: key, stmF: method(c.StmF), strF: method(c.StrF)}
}

// hash2B computes the password hash. For R5 it is a single SHA-256; for R6
// it is Algorithm 2.B: an iterated SHA-256/384/512 and AES-128-CBC mixing
// loop that runs at least 64 rounds. udata is empty for user-password
// hashing and the 48-byte /U for owner-password hashing.
func hash2B(password, salt, udata []byte, r int) []byte {
	h := sha256.New()
	h.Write(password)
	h.Write(salt)
	h.Write(udata)
	k := h.Sum(nil)
	if r < 6 {
		return k
	}

	for round := 0; ; round++ {
		// K1 = (password ‖ K ‖ udata) repeated 64 times.
		block := make([]byte, 0, len(password)+len(k)+len(udata))
		block = append(block, password...)
		block = append(block, k...)
		block = append(block, udata...)
		k1 := bytes.Repeat(block, 64)

		// E = AES-128-CBC encrypt of K1 with key K[0:16], IV K[16:32].
		cb, err := aes.NewCipher(k[:16])
		if err != nil {
			return k[:32]
		}
		e := make([]byte, len(k1))
		cipher.NewCBCEncrypter(cb, k[16:32]).CryptBlocks(e, k1)

		// Next hash chosen by (sum of first 16 bytes) mod 3.
		var sum int
		for _, b := range e[:16] {
			sum += int(b)
		}
		var hh hash.Hash
		switch sum % 3 {
		case 0:
			hh = sha256.New()
		case 1:
			hh = sha512.New384()
		default:
			hh = sha512.New()
		}
		hh.Write(e)
		k = hh.Sum(nil)

		if round >= 63 && int(e[len(e)-1]) <= round-32 {
			break
		}
	}
	return k[:32]
}

// aesNoPadCBC decrypts a wrapped key: AES-256-CBC with a zero IV and no
// padding (ISO 32000-2 §7.6.4.3.4). It returns nil on a malformed input.
func aesNoPadCBC(key, data []byte) []byte {
	if len(key) < 32 || len(data) < 32 || len(data)%aes.BlockSize != 0 {
		return nil
	}
	block, err := aes.NewCipher(key[:32])
	if err != nil {
		return nil
	}
	out := make([]byte, len(data))
	cipher.NewCBCDecrypter(block, make([]byte, aes.BlockSize)).CryptBlocks(out, data)
	return out[:32]
}
