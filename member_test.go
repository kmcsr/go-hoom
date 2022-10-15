
package hoom_test

import (
	"testing"

	"github.com/kmcsr/go-pio/encoding"
	. "github.com/kmcsr/go-hoom"
)

func TestMember(t *testing.T){
	var err error
	member, err := NoAuthMemberServer.QueryMember("0xab")
	if err != nil {
		t.Fatalf("QueryMember: %v", err)
	}
	if member.Id() != "0xab" {
		t.Errorf("member.Id() should be \"0xab\"")
	}
	if member.Name() != "User 0xab" {
		t.Errorf("member.Id() should be \"User 0xab\" but it's \"%s\"", member.Name())
	}
	buf := encoding.NewBuffer(nil)
	if err = NoAuthMemberServer.WriteMember(buf, member); err != nil {
		t.Fatalf("Member.WriteTo: %v", err)
	}
	var member2 *Member
	if member2, err = NoAuthMemberServer.ParseMember(buf); err != nil {
		t.Fatalf("Member.ParseFrom: %v", err)
	}
	if member.Id() != member2.Id() {
		t.Errorf("member.Id() should as same as member2.Id()")
	}
	if member.Name() != member2.Name() {
		t.Errorf("member.Name() should as same as member2.Name()")
	}
}
