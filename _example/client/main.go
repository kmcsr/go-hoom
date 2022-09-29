
package main

import (
	"fmt"
	"io"
	"net"

	hoom "github.com/kmcsr/go-hoom"
)

var localhost = net.IPv4(127, 0, 0, 1)

const roomid = 0x01

func main(){
	target := &net.TCPAddr{IP: localhost, Port: 12348}
	user := hoom.LogMember(0x02, "example-user")
	client, err := user.Dial(target)
	must(err)
	defer client.Close()
	fmt.Println("Client connected", target)
	ping, err := client.Ping()
	must(err)
	fmt.Println("Ping:", ping)

	err = client.Join(roomid)
	must(err)
	fmt.Println("Joined room", roomid)

	listener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: localhost, Port: 12347})
	must(err)
	fmt.Println("Listening:", listener.Addr())
	var conn net.Conn
	for {
		conn, err = listener.Accept()
		must(err)
		go func(conn net.Conn){
			rwc, err := client.Dial(roomid)
			must(err)
			fmt.Printf("client %v dialed\n", conn.RemoteAddr())
			go func(){
				defer rwc.Close()
				defer conn.Close()
				io.Copy(conn, rwc)
			}()
			go func(){
				defer rwc.Close()
				defer conn.Close()
				io.Copy(rwc, conn)
			}()
		}(conn)
	}
}

func must(err error){
	if err != nil {
		panic(err)
	}
}
