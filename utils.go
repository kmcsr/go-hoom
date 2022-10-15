
package hoom

import (
	crand "crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/binary"
	"io"
	"math/big"
	"sync"

	"github.com/kmcsr/go-pio/encoding"
)

func xorBytes(dst, src []byte){
	if len(dst) < len(src) {
		panic("len(dst) < len(src)")
	}
	for i, b := range src {
		dst[i] ^= b
	}
}

func randUint64()(v uint64){
	var buf [8]byte
	_, err := io.ReadFull(crand.Reader, buf[:])
	if err != nil {
		panic(err)
	}
	v = binary.BigEndian.Uint64(buf[:])
	return
}

func rsaWritePubKeyTo(iw io.Writer, key *rsa.PublicKey)(err error){
	w := encoding.WrapWriter(iw)
	if err = w.WriteBytes(key.N.Bytes()); err != nil {
		return
	}
	if err = w.WriteUint64((uint64)(key.E)); err != nil {
		return
	}
	return
}

func rsaReadPubKeyFrom(ir io.Reader)(key *rsa.PublicKey, err error){
	r := encoding.WrapReader(ir)
	var (
		pubN []byte
		pubE uint64
	)
	if pubN, err = r.ReadBytes(); err != nil {
		return
	}
	if pubE, err = r.ReadUint64(); err != nil {
		return
	}
	return &rsa.PublicKey{
		N: new(big.Int).SetBytes(pubN),
		E: (int)(pubE),
	}, nil
}

func rsaEncryptOAEP(key *rsa.PublicKey, data []byte, label []byte)([]byte, error){
	return rsa.EncryptOAEP(sha256.New(), crand.Reader, key, data, label)
}

func rsaDecryptOAEP(key *rsa.PrivateKey, ciphertext []byte, label []byte)([]byte, error){
	return rsa.DecryptOAEP(sha256.New(), crand.Reader, key, ciphertext, label)
}

func rsaEncryptOAEPTo(iw io.Writer, key *rsa.PublicKey, data []byte, label []byte)(err error){
	w := encoding.WrapWriter(iw)
	var cipher []byte
	if cipher, err = rsaEncryptOAEP(key, data, label); err != nil {
		return
	}
	if err = w.WriteBytes(cipher); err != nil {
		return
	}
	return
}

func rsaDecryptOAEPFrom(ir io.Reader, key *rsa.PrivateKey, label []byte)(data []byte, err error){
	r := encoding.WrapReader(ir)
	var cipher []byte
	if cipher, err = r.ReadBytes(); err != nil {
		return
	}
	if data, err = rsaDecryptOAEP(key, cipher, label); err != nil {
		return
	}
	return
}

const copyBufferSize = 1024 * 32

var bufPool = &sync.Pool{
	New: func()(any){
		return make([]byte, copyBufferSize)
	},
}

func ioProxy(a, b io.ReadWriteCloser)(done <-chan error){
	done0 := make(chan error, 1)
	buf1 := bufPool.Get().([]byte)
	go func(){
		defer bufPool.Put(buf1)
		defer a.Close()
		defer b.Close()
		_, err := io.CopyBuffer(b, a, buf1)
		select {
			case done0 <- err:
			default:
		}
	}()
	buf2 := bufPool.Get().([]byte)
	go func(){
		defer bufPool.Put(buf2)
		defer a.Close()
		defer b.Close()
		_, err := io.CopyBuffer(a, b, buf2)
		select {
			case done0 <- err:
			default:
		}
	}()
	return done0
}
