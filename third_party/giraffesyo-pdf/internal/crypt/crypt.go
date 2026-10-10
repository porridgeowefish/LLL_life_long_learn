// Package crypt implements the PDF standard security handler (ISO 32000-1
// §7.6.3 and ISO 32000-2 §7.6.4): RC4 and AES stream/string decryption
// with the file key derived from a supplied user or owner password.
//
// The package is value-agnostic: callers extract the /Encrypt dictionary
// and the /ID into a Config.
package crypt

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5" //nolint:gosec // ISO 32000-1 §7.6.3 mandates MD5 for key derivation
	"crypto/rc4" //nolint:gosec // ISO 32000-1 §7.6.3 mandates RC4 as a stream cipher
	"errors"
	"fmt"
	"io"

	"github.com/giraffesyo/pdf/internal/safeio"
)

// ErrPasswordRequired means the supplied password matched neither the user
// nor the owner password.
var ErrPasswordRequired = errors.New("pdf: encrypted document requires a password")

// Method is the cipher applied to a string or stream.
type Method int

// Cipher methods a crypt filter can select.
const (
	Identity Method = iota // no encryption (the Identity crypt filter)
	RC4
	AESV2 // AES-128-CBC
	AESV3 // AES-256-CBC
)

// Config carries the /Encrypt dictionary fields the handler needs, already
// extracted from PDF objects by the caller.
type Config struct {
	Filter      string // /Filter, must be "Standard"
	V, R        int    // /V algorithm, /R revision
	Length      int    // /Length in bits (default 40)
	O, U        []byte // /O, /U password strings
	OE, UE      []byte // /OE, /UE (R6)
	P           int32  // /P permissions
	EncryptMeta bool   // /EncryptMetadata (default true)
	StmF, StrF  string // /StmF, /StrF crypt filter names (V≥4)
	CF          map[string]Filter
	ID          []byte // first element of the trailer /ID array
}

// Filter is one entry of the /CF crypt-filter dictionary.
type Filter struct {
	CFM    string // /CFM: V2, AESV2, AESV3, or Identity
	Length int    // /Length in bytes (crypt-filter convention)
}

// A Decryptor holds the derived file key and per-string/stream methods.
type Decryptor struct {
	key  []byte
	stmF Method
	strF Method
}

var passwordPad = []byte{
	0x28, 0xBF, 0x4E, 0x5E, 0x4E, 0x75, 0x8A, 0x41,
	0x64, 0x00, 0x4E, 0x56, 0xFF, 0xFA, 0x01, 0x08,
	0x2E, 0x2E, 0x00, 0xB6, 0xD0, 0x68, 0x3E, 0x80,
	0x2F, 0x0C, 0xA9, 0xFE, 0x64, 0x53, 0x69, 0x7A,
}

// New authenticates the empty password and derives the file key.
func New(c Config) (*Decryptor, error) {
	return NewWithPassword(c, nil)
}

// NewWithPassword authenticates password as either the user or owner password
// and derives the file key.
//
// A document whose crypt filters leave strings and streams unencrypted
// (both /StmF and /StrF Identity) encrypts only its embedded files, under
// /EFF. Its content needs no key, so it opens without the password.
func NewWithPassword(c Config, password []byte) (*Decryptor, error) {
	if c.Filter != "" && c.Filter != "Standard" {
		return nil, fmt.Errorf("pdf: unsupported security handler /%s", c.Filter)
	}
	var d *Decryptor
	var err error
	switch {
	case c.V >= 5 || c.R >= 5:
		d, err = newAES256(c, password)
	case c.V >= 1 && c.V <= 4:
		d, err = newRC4orAES(c, password)
	default:
		return nil, fmt.Errorf("pdf: unsupported /Encrypt version V=%d R=%d", c.V, c.R)
	}
	if errors.Is(err, ErrPasswordRequired) && c.V >= 4 &&
		methodForFilter(c, c.StmF) == Identity && methodForFilter(c, c.StrF) == Identity {
		return &Decryptor{stmF: Identity, strF: Identity}, nil
	}
	return d, err
}

// DecryptString decrypts a literal or hex string found in object (num,gen).
func (d *Decryptor) DecryptString(num, gen int, s []byte) []byte {
	return d.crypt(d.strF, num, gen, s)
}

// DecryptStreamData decrypts a fully-read stream body.
func (d *Decryptor) DecryptStreamData(num, gen int, data []byte) []byte {
	return d.crypt(d.stmF, num, gen, data)
}

// DecryptStream wraps r so the stream body decrypts as it is read.
func (d *Decryptor) DecryptStream(num, gen int, r io.Reader) io.Reader {
	// Stream bodies are already size-guarded by callers via safeio; decrypt
	// eagerly to keep the CBC/RC4 handling simple and allocation-bounded.
	data := safeio.ReadAllGuarded(r)
	return bytes.NewReader(d.DecryptStreamData(num, gen, data))
}

func (d *Decryptor) crypt(m Method, num, gen int, data []byte) []byte {
	switch m {
	case Identity:
		return data
	case AESV3:
		return aesDecrypt(d.key, data)
	case AESV2:
		return aesDecrypt(objKey(d.key, num, gen, true), data)
	case RC4:
		return rc4Crypt(objKey(d.key, num, gen, false), data)
	default:
		return data
	}
}

// objKey derives the per-object key (Algorithm 1): MD5 of the file key
// followed by the low 3 bytes of the object number, the low 2 bytes of
// the generation, and — for AES — the "sAlT" constant, truncated to
// len(key)+5 bytes (capped at 16). AES-256 uses the file key directly and
// never calls this.
func objKey(fileKey []byte, num, gen int, aesv2 bool) []byte {
	h := md5.New() //nolint:gosec // ISO 32000-1 §7.6.3 Algorithm 1
	h.Write(fileKey)
	h.Write([]byte{
		byte(num & 0xFF), byte(num >> 8 & 0xFF), byte(num >> 16 & 0xFF),
		byte(gen & 0xFF), byte(gen >> 8 & 0xFF),
	})
	if aesv2 {
		h.Write([]byte{0x73, 0x41, 0x6C, 0x54}) // "sAlT"
	}
	sum := h.Sum(nil)
	n := min(len(fileKey)+5, 16)
	return sum[:n]
}

func rc4Crypt(key, data []byte) []byte {
	c, err := rc4.NewCipher(key) //nolint:gosec // ISO 32000-1 §7.6.3
	if err != nil {
		return data
	}
	out := make([]byte, len(data))
	c.XORKeyStream(out, data)
	return out
}

// aesDecrypt decrypts AES-CBC data whose 16-byte IV prefixes the
// ciphertext, stripping PKCS#7 padding when the final block is valid.
func aesDecrypt(key, data []byte) []byte {
	block, err := aes.NewCipher(key)
	if err != nil || len(data) < aes.BlockSize {
		return nil
	}
	iv, ct := data[:aes.BlockSize], data[aes.BlockSize:]
	if len(ct)%aes.BlockSize != 0 {
		ct = ct[:len(ct)-len(ct)%aes.BlockSize]
	}
	out := make([]byte, len(ct))
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(out, ct)
	if n := len(out); n > 0 {
		if pad := int(out[n-1]); pad >= 1 && pad <= aes.BlockSize && pad <= n {
			out = out[:n-pad]
		}
	}
	return out
}

// hashN is MD5 padded/truncated to n bytes, the shape both key derivation
// and RC4 iterations need.
func hashN(data []byte, n int) []byte {
	sum := md5.Sum(data) //nolint:gosec // ISO 32000-1 §7.6.3
	if n > len(sum) {
		n = len(sum)
	}
	out := make([]byte, n)
	copy(out, sum[:n])
	return out
}
