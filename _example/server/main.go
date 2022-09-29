
package main

import (
	"net"

	hoom "github.com/kmcsr/go-hoom"
)

var localhost = net.IPv4(127, 0, 0, 1)
var server_addr = &net.TCPAddr{IP: localhost, Port: 12348}

func main(){
	owner := hoom.LogMember(0x01, "example-owner")
	server := owner.NewServer(server_addr)
	room := server.NewRoom("example-room", &net.TCPAddr{IP: localhost, Port: 25565})
	println("new room id:", room.Id())
	println("server.ListenAndServe")
	server.ListenAndServe()
}
