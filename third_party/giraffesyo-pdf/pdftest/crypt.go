package pdftest

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5" //nolint:gosec // PDF standard security handler mandates MD5 (ISO 32000-1 §7.6)
	"crypto/rc4" //nolint:gosec // PDF standard security handler mandates RC4 (ISO 32000-1 §7.6)
	"fmt"
	"strings"
)

// EncryptSpec selects the standard-security-handler variant for
// BuildEncrypted.
type EncryptSpec struct {
	R             int  // revision: 2, 3, or 4
	AES           bool // R4 only: AES-128 (AESV2) instead of RC4
	UserPassword  string
	OwnerPassword string
}

var padString = []byte{
	0x28, 0xBF, 0x4E, 0x5E, 0x4E, 0x75, 0x8A, 0x41,
	0x64, 0x00, 0x4E, 0x56, 0xFF, 0xFA, 0x01, 0x08,
	0x2E, 0x2E, 0x00, 0xB6, 0xD0, 0x68, 0x3E, 0x80,
	0x2F, 0x0C, 0xA9, 0xFE, 0x64, 0x53, 0x69, 0x7A,
}

// BuildEncrypted assembles an encrypted classic-xref PDF. objs[i] becomes
// object i+1; stream bodies are encrypted with the per-object key. The
// /Encrypt dictionary is added as an extra object. Fixtures must not
// contain literal strings outside streams (the builder does not rewrite
// them). rootID is the catalog.
func BuildEncrypted(rootID int, spec EncryptSpec, objs ...string) []byte {
	const keyLen = 16 // 128-bit for R3/4; R2 uses 40-bit below
	id := []byte("0123456789abcdef")
	p := int32(-44)

	kl := keyLen
	if spec.R == 2 {
		kl = 5
	}
	o := ownerString(kl, spec.R, []byte(spec.OwnerPassword), []byte(spec.UserPassword))
	fileKey := deriveFileKey(o, p, id, kl, spec.R, []byte(spec.UserPassword))
	u := userString(fileKey, id, spec.R)

	encNum := len(objs) + 1
	encDict := encryptDict(spec, o, u, p, kl)

	var b bytes.Buffer
	b.WriteString("%PDF-1.5\n")
	offsets := make([]int, len(objs)+2)
	for i, obj := range objs {
		offsets[i+1] = b.Len()
		enc := encryptStreams(obj, i+1, fileKey, spec)
		fmt.Fprintf(&b, "%d 0 obj\n%s\nendobj\n", i+1, enc)
	}
	offsets[encNum] = b.Len()
	fmt.Fprintf(&b, "%d 0 obj\n%s\nendobj\n", encNum, encDict)

	xref := b.Len()
	fmt.Fprintf(&b, "xref\n0 %d\n0000000000 65535 f \n", encNum+1)
	for i := 1; i <= encNum; i++ {
		fmt.Fprintf(&b, "%010d 00000 n \n", offsets[i])
	}
	fmt.Fprintf(&b, "trailer\n<< /Size %d /Root %d 0 R /Encrypt %d 0 R /ID [<%X> <%X>] >>\nstartxref\n%d\n%%%%EOF\n",
		encNum+1, rootID, encNum, id, id, xref)
	return b.Bytes()
}

func encryptDict(spec EncryptSpec, o, u []byte, p int32, keyLen int) string {
	switch {
	case spec.R == 2:
		return fmt.Sprintf("<< /Filter /Standard /V 1 /R 2 /O <%X> /U <%X> /P %d >>", o, u, p)
	case spec.R == 4 && spec.AES:
		return fmt.Sprintf("<< /Filter /Standard /V 4 /R 4 /Length %d /O <%X> /U <%X> /P %d "+
			"/CF << /StdCF << /CFM /AESV2 /Length %d >> >> /StmF /StdCF /StrF /StdCF >>",
			keyLen*8, o, u, p, keyLen)
	case spec.R == 4:
		return fmt.Sprintf("<< /Filter /Standard /V 4 /R 4 /Length %d /O <%X> /U <%X> /P %d "+
			"/CF << /StdCF << /CFM /V2 /Length %d >> >> /StmF /StdCF /StrF /StdCF >>",
			keyLen*8, o, u, p, keyLen)
	default:
		return fmt.Sprintf("<< /Filter /Standard /V 2 /R 3 /Length %d /O <%X> /U <%X> /P %d >>",
			keyLen*8, o, u, p)
	}
}

func pad(pw []byte) []byte {
	out := make([]byte, 32)
	n := copy(out, pw)
	copy(out[n:], padString)
	return out
}

func hashN(data []byte, n int) []byte {
	sum := md5.Sum(data) //nolint:gosec // ISO 32000-1 §7.6.3
	out := make([]byte, n)
	copy(out, sum[:n])
	return out
}

func rc4Bytes(key, data []byte) []byte {
	c, _ := rc4.NewCipher(key) //nolint:gosec // ISO 32000-1 §7.6.3
	out := make([]byte, len(data))
	c.XORKeyStream(out, data)
	return out
}

func ownerString(keyLen, r int, ownerPassword, userPassword []byte) []byte {
	key := hashN(pad(ownerPassword), 16)
	if r >= 3 {
		for range 50 {
			key = hashN(key, 16)
		}
	}
	key = key[:keyLen]
	o := pad(userPassword)
	if r == 2 {
		return rc4Bytes(key, o)
	}
	for i := range 20 {
		o = rc4Bytes(xor(key, byte(i)), o)
	}
	return o
}

func deriveFileKey(o []byte, p int32, id []byte, keyLen, r int, userPassword []byte) []byte {
	h := md5.New() //nolint:gosec // ISO 32000-1 §7.6.3 Algorithm 2
	h.Write(pad(userPassword))
	h.Write(o)
	up := uint32(p) //nolint:gosec // /P is a 32-bit flag field
	h.Write([]byte{byte(up & 0xFF), byte(up >> 8 & 0xFF), byte(up >> 16 & 0xFF), byte(up >> 24 & 0xFF)})
	h.Write(id)
	key := h.Sum(nil)[:keyLen]
	if r >= 3 {
		for range 50 {
			key = hashN(key, keyLen)
		}
	}
	return key
}

func userString(fileKey, id []byte, r int) []byte {
	if r == 2 {
		return rc4Bytes(fileKey, padString)
	}
	h := md5.New() //nolint:gosec // ISO 32000-1 §7.6.3 Algorithm 5
	h.Write(padString)
	h.Write(id)
	digest := rc4Bytes(fileKey, h.Sum(nil))
	for i := 1; i <= 19; i++ {
		digest = rc4Bytes(xor(fileKey, byte(i)), digest)
	}
	out := make([]byte, 32)
	copy(out, digest[:16])
	return out
}

func objectKey(fileKey []byte, num int, aesV2 bool) []byte {
	h := md5.New() //nolint:gosec // ISO 32000-1 §7.6.3 Algorithm 1
	h.Write(fileKey)
	h.Write([]byte{byte(num & 0xFF), byte(num >> 8 & 0xFF), byte(num >> 16 & 0xFF), 0, 0})
	if aesV2 {
		h.Write([]byte{0x73, 0x41, 0x6C, 0x54})
	}
	n := min(len(fileKey)+5, 16)
	return h.Sum(nil)[:n]
}

func aesEncryptBytes(key, data []byte) []byte {
	block, _ := aes.NewCipher(key)
	iv := bytes.Repeat([]byte{0x24}, aes.BlockSize)
	padLen := aes.BlockSize - len(data)%aes.BlockSize
	padded := append(append([]byte{}, data...), bytes.Repeat([]byte{byte(padLen)}, padLen)...)
	ct := make([]byte, len(padded))
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(ct, padded)
	return append(append([]byte{}, iv...), ct...)
}

// encryptStreams rewrites the "stream\n…\nendstream" body of an object,
// encrypting the data with the per-object key. Objects with no stream are
// returned unchanged.
func encryptStreams(obj string, num int, fileKey []byte, spec EncryptSpec) string {
	i := strings.Index(obj, "stream\n")
	j := strings.LastIndex(obj, "\nendstream")
	if i < 0 || j < 0 || j < i {
		return obj
	}
	dataStart := i + len("stream\n")
	body := []byte(obj[dataStart:j])

	var enc []byte
	if spec.R == 4 && spec.AES {
		enc = aesEncryptBytes(objectKey(fileKey, num, true), body)
	} else {
		enc = rc4Bytes(objectKey(fileKey, num, false), body)
	}

	// Rewrite /Length to the ciphertext length.
	head := obj[:i]
	head = replaceLength(head, len(enc))
	return head + "stream\n" + string(enc) + obj[j:]
}

// replaceLength rewrites the "/Length N" token in a stream dict header.
func replaceLength(head string, n int) string {
	k := strings.Index(head, "/Length")
	if k < 0 {
		return head
	}
	end := k + len("/Length")
	for end < len(head) && head[end] == ' ' {
		end++
	}
	numEnd := end
	for numEnd < len(head) && head[numEnd] >= '0' && head[numEnd] <= '9' {
		numEnd++
	}
	return head[:k] + fmt.Sprintf("/Length %d", n) + head[numEnd:]
}

func xor(key []byte, x byte) []byte {
	out := make([]byte, len(key))
	for i, b := range key {
		out[i] = b ^ x
	}
	return out
}
