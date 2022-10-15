
package hoom

import (
	"bytes"
	"io"
	"net"

	"github.com/kmcsr/go-pio/encoding"
)

type network = uint8

const (
	_ network = 0x00
	networkTCP = 0x01
	networkUDP = 0x02
)

type ConnToken interface{
	Target()(net.Addr)
	PublicKey()([]byte)
	Encode()([]byte)
}

type connToken struct{
	target net.Addr
	pubKey []byte
}

func ParseConnToken(buf []byte)(_ ConnToken, err error){
	var t connToken
	r := encoding.WrapReader(bytes.NewReader(buf))
	var netw network
	if netw, err = r.ReadByte(); err != nil {
		return
	}
	switch netw {
	case networkTCP:
		var (
			ip []byte
			port uint16
		)
		if ip, err = readIP(r); err != nil {
			return
		}
		if port, err = r.ReadUint16(); err != nil {
			return
		}
		t.target = &net.TCPAddr{
			IP: (net.IP)(ip),
			Port: (int)(port),
		}
	case networkUDP:
		var (
			ip []byte
			port uint16
		)
		if ip, err = readIP(r); err != nil {
			return
		}
		if port, err = r.ReadUint16(); err != nil {
			return
		}
		t.target = &net.UDPAddr{
			IP: (net.IP)(ip),
			Port: (int)(port),
		}
	default:
		panic("unknown network id")
	}
	var flag bool
	if flag, err = r.ReadBool(); err != nil {
		return
	}
	if flag {
		if t.pubKey, err = r.ReadBytes(); err != nil {
			return
		}
	}
	return t, nil
}

func (t connToken)Target()(net.Addr){
	return t.target
}

func (t connToken)PublicKey()([]byte){
	return t.pubKey
}

func (t connToken)Encode()([]byte){
	buf := encoding.NewBuffer(nil)
	switch addr := t.target.(type) {
	case *net.TCPAddr:
		buf.WriteByte(networkTCP)
		writeIP(buf, addr.IP)
		buf.WriteUint16((uint16)(addr.Port))
	case *net.UDPAddr:
		buf.WriteByte(networkUDP)
		writeIP(buf, addr.IP)
		buf.WriteUint16((uint16)(addr.Port))
	default:
		panic("unknown type of net.Addr")
	}
	if len(t.pubKey) > 0 {
		buf.WriteBool(true)
		buf.WriteBytes(t.pubKey)
	}else{
		buf.WriteBool(false)
	}
	return buf.Bytes()
}

func writeIP(w encoding.Writer, ip net.IP)(err error){
	if ip4 := ip.To4(); ip4 != nil {
		ip = ip4
	}
	if err = w.WriteByte((byte)(len(ip))); err != nil {
		return
	}
	if _, err = w.Write(([]byte)(ip)); err != nil {
		return
	}
	return
}

func readIP(r encoding.Reader)(ip net.IP, err error){
	var l byte
	if l, err = r.ReadByte(); err != nil {
		return
	}
	ip0 := make([]byte, l)
	if _, err = io.ReadFull(r, ip0); err != nil {
		return
	}
	ip = (net.IP)(ip0)
	return
}

type DialConfig struct{
	Target net.Addr
	Token ConnToken
	handshakers map[byte]Handshaker
}

func DialAddr(addr net.Addr)(*DialConfig){
	return &DialConfig{
		Target: addr,
	}
}

func DialToken(t ConnToken)(*DialConfig){
	return &DialConfig{
		Target: t.Target(),
		Token: t,
	}
}

func (c *DialConfig)AddHandshaker(h Handshaker)(*DialConfig){
	if c.handshakers == nil {
		c.handshakers = make(map[byte]Handshaker)
	}
	c.handshakers[h.Id()] = h
	return c
}

func (c *DialConfig)GetHandshaker(id byte)(hs Handshaker, ok bool){
	if c.handshakers == nil {
		return nil, false
	}
	hs, ok = c.handshakers[id]
	return
}

func (c *DialConfig)PopHandshaker(id byte){
	delete(c.handshakers, id)
}

func (c *DialConfig)Handshakers()(hs []Handshaker){
	if c.handshakers == nil {
		return nil
	}
	hs = make([]Handshaker, 0, len(c.handshakers))
	for _, h := range c.handshakers {
		hs = append(hs, h)
	}
	return
}

func (cfg *DialConfig)handshake(c io.ReadWriteCloser)(rw io.ReadWriteCloser, err error){
	r := encoding.WrapReader(c)
	w := encoding.WrapWriter(c)
	loger.Tracef("AuthedMember.handshakers: %v", cfg.handshakers)
	var id byte
	for {
		if id, err = r.ReadByte(); err != nil {
			return
		}
		loger.Tracef("hoom.Client: Trying handshaker 0x%02x", id)
		if id == NoneConnId {
			break
		}
		hs, ok := cfg.GetHandshaker(id)
		if err = w.WriteBool(ok); err != nil {
			return
		}
		if ok {
			return hs.HandClient(c, cfg)
		}
	}
	return nil, NoCommonHandshaker
}
