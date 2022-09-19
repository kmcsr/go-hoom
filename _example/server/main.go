
package main

import (
	"net"

	hoom "github.com/kmcsr/go-hoom"
)

var localhost = net.IPv4(127, 0, 0, 1)
var server_addr = &net.TCPAddr{IP: localhost, Port: 12348}

const roomid = 0xab

func main(){
	owner := hoom.NewMember(0x01, "example-owner")
	server := hoom.NewServer(server_addr, owner)
	room := hoom.NewRoom(roomid, "example-room", server, &net.TCPAddr{IP: localhost, Port: 25565})
	_  = room
	println("server.ListenAndServe")
	server.ListenAndServe()
}
