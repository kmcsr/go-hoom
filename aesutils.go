
package hoom

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	crand "crypto/rand"
	"io"
	"math/rand"
	"sync"

	"github.com/kmcsr/go-pio/encoding"
)

func genAesKey()(key []byte, err error){
	key = make([]byte, 32) // 256-bit
	if _, err = io.ReadFull(crand.Reader, key); err != nil {
		key = nil
		return
	}
	return
}

type aesStream struct{
	cipher.Block
	rd           *rand.Rand
	lastR, lastW []byte
	rl, wl       sync.Mutex
	r            io.Reader
	w            io.Writer

	rbuf []byte
}

func newAesStream(key []byte, r io.Reader, w io.Writer)(s *aesStream, err error){
	s = new(aesStream)
	s.r = r
	s.w = w
	if s.Block, err = aes.NewCipher(key); err != nil {
		return
	}
	bs := s.BlockSize()
	s.lastR = make([]byte, bs)
	s.lastW = make([]byte, bs)
	seed := (int64)(encoding.DecodeUint64(key[0:8]))
	s.rd = rand.New(rand.NewSource(seed))
	if _, err = io.ReadFull(s.rd, s.lastR); err != nil {
		return
	}
	copy(s.lastW, s.lastR)
	return
}

func (s *aesStream)Close()(err error){
	if c, ok := s.r.(io.Closer); ok {
		if er := c.Close(); er != nil {
			err = er
		}
	}
	if c, ok := s.w.(io.Closer); ok {
		if er := c.Close(); er != nil && err == nil {
			err = er
		}
	}
	return
}

func (s *aesStream)Write(buf []byte)(n int, err error) {
	if len(buf) == 0 {
		return
	}
	s.wl.Lock()
	defer s.wl.Unlock()

	bs := s.BlockSize()
	bf := make([]byte, bs)
	if len(buf) < bs {
		bf[0] = (byte)(copy(bf[1:], buf))
		s.xorEncrypt(bf, bf)
		if _, err = s.w.Write(bf); err != nil {
			return
		}
		return len(buf), nil
	}
	bts := bytes.NewBuffer(nil)
	bf[0] = 0x00
	encoding.EncodeUint32(bf[1:1+8], (uint32)(len(buf)))
	s.xorEncrypt(bf, bf)
	bts.Write(bf)
	for i, j := 0, bs; i < len(buf); i, j = j, j+bs {
		if j > len(buf) {
			copy(bf, buf[i:])
			s.xorEncrypt(bf, bf)
			bts.Write(bf)
			break
		}
		s.xorEncrypt(bf, buf[i:j])
		bts.Write(bf)
	}
	if _, err = bts.WriteTo(s.w); err != nil {
		return
	}
	return len(buf), nil
}

func (s *aesStream)xorEncrypt(dst, src []byte){
	xorBytes(src, s.lastW)
	s.Encrypt(dst, src)
	xorBytes(s.lastW, dst)
}

func (s *aesStream)Read(buf []byte)(n int, err error){
	if len(buf) == 0 {
		return
	}
	s.rl.Lock()
	defer s.rl.Unlock()

	if len(s.rbuf) > 0 {
		n = copy(buf, s.rbuf)
		s.rbuf = s.rbuf[n:]
		return
	}

	bs := s.BlockSize()
	bf := make([]byte, bs)
	if _, err = io.ReadFull(s.r, bf); err != nil {
		return
	}
	s.xorDecrypt(bf, bf)
	if l := (int)(bf[0]); l != 0x00 {
		n = copy(buf, bf[1:1+l])
		if n < l {
			s.rbuf = append(s.rbuf, bf[1 + n:1 + l]...)
		}
		return
	}
	l := (int)(encoding.DecodeUint32(bf[1:1 + 8]))
	bts := make([]byte, l)
	for i, j := 0, bs; i < l; i, j = j, j+bs {
		if _, err = io.ReadFull(s.r, bf); err != nil {
			return
		}
		s.xorDecrypt(bf, bf)
		if j > l {
			copy(bts[i:l], bf[:l - i])
			break
		}
		copy(bts[i:j], bf)
	}
	n = copy(buf, bts)
	if n < len(bts){
		s.rbuf = append(s.rbuf, bts[n:]...)
	}
	return
}

func (s *aesStream)xorDecrypt(dst, src []byte){
	last := append(([]byte)(nil), s.lastR...)
	xorBytes(last, src)
	s.Decrypt(dst, src)
	xorBytes(dst, s.lastR)
	copy(s.lastR, last)
}
