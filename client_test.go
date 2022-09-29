
package hoom_test

import (
	"testing"

	. "github.com/kmcsr/go-hoom"
)

func TestClient(t *testing.T){
	mem := LogMember(0x33, "client-user")
	_ = mem
	// client, err := mem.Dial(&net.TCPAddr{IP: localhost, Port: port})
	// if err != nil {
	// 	t.Errorf("Member.Dial: %v", err)
	// 	return
	// }
}
