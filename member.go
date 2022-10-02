
package hoom

import "fmt"
import (
	"github.com/kmcsr/go-pio/encoding"
	hdata "github.com/kmcsr/go-hoom/data"
)

type Member = hdata.Member

var NewMember = hdata.NewMember

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
	m = NewMember(id, fmt.Sprintf("user-%d", id))
	return
}

type AuthedMember struct{
	*Member
	authToken string
}

func LogMember(id uint32, token string)(m *AuthedMember, err error){
	var m0 *Member
	if m0, err = QueryMember(id); err != nil {
		return
	}
	// TODO: auth member
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
