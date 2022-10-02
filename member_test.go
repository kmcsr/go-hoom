
package hoom_test

import (
	"bytes"
	"testing"

	"github.com/kmcsr/go-pio/encoding"
	. "github.com/kmcsr/go-hoom"
)

func TestMember(t *testing.T){
	var err error
	member, err := QueryMember(0xab)
	if err != nil {
		t.Fatalf("QueryMember: %v", err)
	}
	if member.Id() != 0xab {
		t.Errorf("member.Id() should be 0xab")
	}
	if member.Name() != "user-171" {
		t.Errorf("member.Id() should be \"user-171\" but it's \"%s\"", member.Name())
	}
	buf := bytes.NewBuffer(nil)
	w := encoding.WrapWriter(buf)
	if err = WriteMember(w, member); err != nil {
		t.Fatalf("Member.WriteTo: %v", err)
	}
	r := encoding.WrapReader(bytes.NewReader(buf.Bytes()))
	var member2 *Member
	if member2, err = ParseMember(r); err != nil {
		t.Fatalf("Member.ParseFrom: %v", err)
	}
	if member.Id() != member2.Id() {
		t.Errorf("member.Id() should as same as member2.Id()")
	}
	if member.Name() != member2.Name() {
		t.Errorf("member.Name() should as same as member2.Name()")
	}
}
