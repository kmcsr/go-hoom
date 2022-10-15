
package hoom_test

import (
	"testing"
	"net"

	. "github.com/kmcsr/go-hoom"
)

var localhost = net.IPv4(127, 0, 0, 1)
var exampleUser = NewMember(0x22, "example-user")

func TestServer(t *testing.T){
	mem, err := LogMember(0x11, "<TOKEN>")
	if err != nil {
		t.Fatalf("Logging error: %v", err)
	}
	server := mem.NewServer(&net.TCPAddr{IP: localhost}).AddHandshaker(UnsafeHandshaker)
	server.Listen()
	defer server.Shutdown()
	addr := server.ListenAddr()
	t.Logf("listening: %v", addr)
	// go server.Serve()
}
