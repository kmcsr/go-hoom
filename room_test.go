
package hoom_test

import (
	"testing"

	"github.com/kmcsr/go-pio/encoding"
	. "github.com/kmcsr/go-hoom"
)

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
	buf := encoding.NewBuffer(nil)
	if err := token.WriteTo(buf); err != nil {
		t.Fatalf("RoomToken.WriteTo: %v", err)
	}
	token2 := new(RoomToken)
	if err := token2.ParseFrom(buf); err != nil {
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
