
package hoom

import (
	"errors"
	// "fmt"
	"io"

	"github.com/kmcsr/go-pio/encoding"
	hdata "github.com/kmcsr/go-hoom/data"
)

type Member = hdata.Member

func NewMember(id string, name string)(*Member){
	return hdata.NewMember(id, name)
}

type MemberServer interface{
	WriteMember(w io.Writer, m *Member)(err error)
	ParseMember(r io.Reader)(m *Member, err error)
	QueryMember(id string)(m *Member, err error)
	AuthMember(id string, password string)(authedMem *AuthedMember, err error)
}

var UserIdNotExists = errors.New("User id not exists")
var PasswordIncorrectErr = errors.New("Password incorrect")

type noAuthMemberServer struct{}

var NoAuthMemberServer MemberServer = noAuthMemberServer{}

func (noAuthMemberServer)WriteMember(iw io.Writer, m *Member)(err error){
	w := encoding.WrapWriter(iw)
	if err = w.WriteString(m.Id()); err != nil {
		return
	}
	return
}

func (s noAuthMemberServer)ParseMember(ir io.Reader)(m *Member, err error){
	r := encoding.WrapReader(ir)
	var id string
	if id, err = r.ReadString(); err != nil {
		return
	}
	if m, err = s.QueryMember(id); err != nil {
		return
	}
	return
}

func (noAuthMemberServer)QueryMember(id string)(m *Member, err error){
	if id == "" {
		return nil, UserIdNotExists
	}
	return NewMember(id, "User " + id), nil
}

func (s noAuthMemberServer)AuthMember(id string, password string)(authedMem *AuthedMember, err error){
	var mem *Member
	if mem, err = s.QueryMember(id); err != nil {
		return
	}
	if password != "" {
		return nil, PasswordIncorrectErr
	}
	return NewAuthMember(s, mem, "AuthToken:noAuthMemberServer"), nil
}


type AuthedMember struct{
	*Member

	authServer MemberServer
	authToken string
}

func NewAuthMember(authServer MemberServer, mem *Member, authToken string)(*AuthedMember){
	return &AuthedMember{
		Member: mem,
		authServer: authServer,
		authToken: authToken,
	}
}

func (m *AuthedMember)MemberServer()(MemberServer){
	return m.authServer
}

var (
	UnsafeConnNotAllowedErr = errors.New("Unsafe connection not allowed")
	PubkeyNotVerifiedErr = errors.New("Public key not verified")
	ConnHijackedErr = errors.New("Connection has been hijacked")
)
