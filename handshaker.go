
package hoom

import (
	"bytes"
	"crypto/rsa"
	"crypto/sha256"
	"errors"
	"io"
	"net"

	"github.com/kmcsr/go-pio/encoding"
)

var NoCommonHandshaker = errors.New("Unable to find a common handshaker")

const (
	NoneConnId byte = 0x00
	UnsafeConnId = 0x01
	RsaConnId    = 0x02
)

type Handshaker interface{
	Id()(byte)

	// Client sided
	HandClient(c io.ReadWriteCloser, config *DialConfig)(rw io.ReadWriteCloser, err error)

	// Server side
	ConnToken(pubaddr net.Addr)(ConnToken)
	HandServer(c io.ReadWriteCloser)(rw io.ReadWriteCloser, err error)
}

type unsafeHandshaker struct{}

var UnsafeHandshaker Handshaker = unsafeHandshaker{}

func (unsafeHandshaker)Id()(byte){
	return UnsafeConnId
}

func (unsafeHandshaker)HandClient(c io.ReadWriteCloser, config *DialConfig)(rw io.ReadWriteCloser, err error){
	return c, nil
}

func (unsafeHandshaker)ConnToken(pubaddr net.Addr)(ConnToken){
	return connToken{
		target: pubaddr,
	}
}

func (unsafeHandshaker)HandServer(c io.ReadWriteCloser)(rw io.ReadWriteCloser, err error){
	return c, nil
}

type RsaHandshaker struct{
	Key *rsa.PrivateKey
}

var _ Handshaker = (*RsaHandshaker)(nil)

var rsaHandshake1Label = ([]byte)("handshake-1")

func (*RsaHandshaker)Id()(byte){
	return RsaConnId
}

func (*RsaHandshaker)HandClient(c io.ReadWriteCloser, config *DialConfig)(rw io.ReadWriteCloser, err error){
	r := encoding.WrapReader(c)
	w := encoding.WrapWriter(c)
	t := config.Token
	var pkey *rsa.PublicKey
	if pkey, err = rsaReadPubKeyFrom(r); err != nil {
		return
	}
	h := sha256.New()
	rsaWritePubKeyTo(h, pkey)
	if !bytes.Equal(t.PublicKey(), h.Sum(nil)) {
		return nil, PubkeyNotVerifiedErr
	}
	var aesKey []byte
	if aesKey, err = genAesKey(); err != nil {
		return
	}
	var (
		rdn uint64 = randUint64()
		rdn2 uint64
	)
	buf := encoding.NewBuffer(nil)
	buf.WriteUint64(rdn)
	buf.WriteBytes(aesKey)
	if err = rsaEncryptOAEPTo(w, pkey, buf.Bytes(), rsaHandshake1Label); err != nil {
		return
	}
	if rw, err = newAesStream(aesKey, c, c); err != nil {
		return
	}
	if rdn2, err = encoding.WrapReader(rw).ReadUint64(); err != nil {
		return
	}
	if rdn2 != rdn {
		return nil, ConnHijackedErr
	}
	return
}

func (s *RsaHandshaker)ConnToken(pubaddr net.Addr)(ConnToken){
	h := sha256.New()
	rsaWritePubKeyTo(h, &s.Key.PublicKey)
	return connToken{
		target: pubaddr,
		pubKey: h.Sum(nil),
	}
}

func (s *RsaHandshaker)HandServer(c io.ReadWriteCloser)(rw io.ReadWriteCloser, err error){
	if err = rsaWritePubKeyTo(c, &s.Key.PublicKey); err != nil {
		return
	}
	var buf []byte
	if buf, err = rsaDecryptOAEPFrom(c, s.Key, rsaHandshake1Label); err != nil {
		return
	}
	rb := encoding.NewBuffer(buf)
	var rdn uint64
	if rdn, err = rb.ReadUint64(); err != nil {
		return
	}
	if buf, err = rb.ReadBytes(); err != nil {
		return
	}
	if rw, err = newAesStream(buf, c, c); err != nil {
		return
	}
	if err = encoding.WrapWriter(rw).WriteUint64(rdn); err != nil {
		return
	}
	return
}
