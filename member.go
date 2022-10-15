
package hoom

import (
	"errors"
	"fmt"
	"io"

	"github.com/kmcsr/go-pio/encoding"
	hdata "github.com/kmcsr/go-hoom/data"
)

type Member = hdata.Member

func NewMember(id uint32, name string)(*Member){
	return hdata.NewMember(id, name)
}

func WriteMember(w encoding.Writer, m *Member)(err error){
	if err = w.WriteUint32(m.Id()); err != nil {
		return
	}
	return
}

func ParseMember(r encoding.Reader)(m *Member, err error){
	var memid uint32
	if memid, err = r.ReadUint32(); err != nil {
		return
	}
	return QueryMember(memid)
}

func QueryMember(id uint32)(m *Member, err error){
	// TODO: query member from global server
	m = NewMember(id, fmt.Sprintf("user-%d", id))
	return
}

type AuthedMember struct{
	*Member
	authToken string
}

func LogMember(id uint32, token string)(m *AuthedMember, err error){
	var m0 *Member
	// TODO: auth member
	m0 = NewMember(id, fmt.Sprintf("user-%d", id))
	authToken := ""
	m = &AuthedMember{
		Member: m0,
		authToken: authToken,
	}
	return
}

func (m *AuthedMember)GetMem()(*Member){
	return m.Member
}

var (
	UnsafeConnNotAllowedErr = errors.New("Unsafe connection not allowed")
	PubkeyNotVerifiedErr = errors.New("Public key not verified")
	ConnHijackedErr = errors.New("Connection has been hijacked")
)

func (m *AuthedMember)handshake(c io.ReadWriteCloser, config *DialConfig)(rw io.ReadWriteCloser, err error){
	r := encoding.WrapReader(c)
	w := encoding.WrapWriter(c)
	loger.Tracef("AuthedMember.handshakers: %v", config.handshakers)
	var id byte
	for {
		if id, err = r.ReadByte(); err != nil {
			return
		}
		loger.Tracef("hoom.Client: Trying handshaker 0x%02x", id)
		if id == NoneConnId {
			break
		}
		hs, ok := config.GetHandshaker(id)
		if err = w.WriteBool(ok); err != nil {
			return
		}
		if ok {
			return hs.HandClient(c, config)
		}
	}
	return nil, NoCommonHandshaker
}
