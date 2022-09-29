
package hoom_test

import (
	"bytes"
	"testing"

	"github.com/kmcsr/go-pio/encoding"
	. "github.com/kmcsr/go-hoom"
)

func TestMember(t *testing.T){
	member := NewMember(0xab, "example-user")
	if member.Id() != 0xab {
		t.Errorf("member.Id() should be 0xab")
	}
	if member.Name() != "example-user" {
		t.Errorf("member.Id() should be \"example-user\"")
	}
	buf := bytes.NewBuffer(nil)
	w := encoding.WrapWriter(buf)
	if err := member.WriteTo(w); err != nil {
		t.Fatalf("Member.WriteTo: %v", err)
	}
	r := encoding.WrapReader(bytes.NewReader(buf.Bytes()))
	member2 := new(Member)
	if err := member2.ParseFrom(r); err != nil {
		t.Fatalf("Member.ParseFrom: %v", err)
	}
	if member.Id() != member2.Id() {
		t.Errorf("member.Id() should as same as member2.Id()")
	}
	if member.Name() != member2.Name() {
		t.Errorf("member.Name() should as same as member2.Name()")
	}
}

func TestRoom(t *testing.T){
	//
}

func TestRoomToken(t *testing.T){
	token := &RoomToken{
		RoomId: 0xabcd,
		MemId: 0xab,
		Token: 0x54321,
		Sign: nil, // TODO: sign token
	}
	buf := bytes.NewBuffer(nil)
	w := encoding.WrapWriter(buf)
	if err := token.WriteTo(w); err != nil {
		t.Fatalf("RoomToken.WriteTo: %v", err)
	}
	r := encoding.WrapReader(bytes.NewReader(buf.Bytes()))
	token2 := new(RoomToken)
	if err := token2.ParseFrom(r); err != nil {
		t.Fatalf("RoomToken.ParseFrom: %v", err)
	}
	if token.RoomId != token2.RoomId {
		t.Errorf("token.RoomId should as same as token2.RoomId")
	}
	if token.MemId != token2.MemId {
		t.Errorf("token.MemId should as same as token2.MemId")
	}
	if token.Token != token2.Token {
		t.Errorf("token.Token should as same as token2.Token")
	}
}
