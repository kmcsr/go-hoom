
package main

import (
	"fmt"
	// "io"
	"net"

	hoom "github.com/kmcsr/go-hoom"
)

var localhost = net.IPv4(127, 0, 0, 1)

const roomid = 0xab

func main(){
	target := &net.TCPAddr{IP: localhost, Port: 12348}
	user := hoom.NewMember(0x02, "example-user")
	client, err := hoom.NewClient(user, target)
	must(err)
	defer client.Close()
	fmt.Println("Client connected", target)
	ping, err := client.Ping()
	must(err)
	fmt.Println("Ping:", ping)

	// err = client.Join(roomid)
	// must(err)
	// fmt.Println("Joined room", roomid)

	listener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: localhost, Port: 12347})
	must(err)
	var conn net.Conn
	for {
		conn, err = listener.Accept()
		must(err)
		go func(conn net.Conn){
			_, done, err := client.Dial(roomid, conn)
			must(err)
			must(<-done)
		}(conn)
	}
}

func must(err error){
	if err != nil {
		panic(err)
	}
}
