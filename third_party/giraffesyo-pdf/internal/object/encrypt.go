package object

import (
	"github.com/giraffesyo/pdf/internal/crypt"
)

// initEncrypt configures the decryptor when the trailer has an /Encrypt
// entry. The /Encrypt dictionary itself is read before the decryptor
// exists, so its strings (/O, /U) are never decrypted; its object number
// is recorded so a later re-resolution stays exempt too.
func (r *Reader) initEncrypt(password []byte) error {
	ev, ok := r.trailer["Encrypt"]
	if !ok {
		return nil
	}
	if ref, ok := ev.(ref); ok {
		r.encNum = ref.num
	}
	enc := Value{r: r, data: ev}
	if enc.Kind() != Dict {
		return nil
	}

	cfg := crypt.Config{
		Filter:      enc.Key("Filter").Name(),
		V:           int(enc.Key("V").intOr(0)),
		R:           int(enc.Key("R").intOr(0)),
		Length:      int(enc.Key("Length").intOr(40)),
		O:           []byte(enc.Key("O").RawString()),
		U:           []byte(enc.Key("U").RawString()),
		OE:          []byte(enc.Key("OE").RawString()),
		UE:          []byte(enc.Key("UE").RawString()),
		P:           int32(enc.Key("P").intOr(0)), //nolint:gosec // /P is a 32-bit flag field
		EncryptMeta: true,
		StmF:        enc.Key("StmF").Name(),
		StrF:        enc.Key("StrF").Name(),
		ID:          r.firstID(),
	}
	if b, ok := enc.Key("EncryptMetadata").Bool(); ok {
		cfg.EncryptMeta = b
	}
	cfg.CF = cryptFilters(enc.Key("CF"))

	dec, err := crypt.NewWithPassword(cfg, password)
	if err != nil {
		return err
	}
	r.dec = dec
	return nil
}

// firstID returns the first element of the trailer /ID array.
func (r *Reader) firstID() []byte {
	id := r.Trailer().Key("ID")
	if id.Kind() != Array || id.Len() == 0 {
		return nil
	}
	return []byte(id.Index(0).RawString())
}

// cryptFilters reads the /CF crypt-filter dictionary.
func cryptFilters(cf Value) map[string]crypt.Filter {
	if cf.Kind() != Dict {
		return nil
	}
	out := map[string]crypt.Filter{}
	for _, key := range cf.Keys() {
		f := cf.Key(key)
		out[key] = crypt.Filter{
			CFM:    f.Key("CFM").Name(),
			Length: int(f.Key("Length").intOr(0)),
		}
	}
	return out
}
