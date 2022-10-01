
package main

import (
	"io"
	"net"
	"os"

  "github.com/sirupsen/logrus"
  "github.com/kmcsr/go-logger"
  logrusl "github.com/kmcsr/go-logger/logrus"
	"github.com/kmcsr/go-hoom"
)

var localhost = net.IPv4(127, 0, 0, 1)

const roomid = 0x01

var loger = initLogger()

func initLogger()(loger logger.Logger){
  loger = logrusl.New()
  loger.SetOutput(os.Stderr)
  logrusl.Unwrap(loger).SetFormatter(&logrus.TextFormatter{
    TimestampFormat: "2006-01-02 15:04:05.000",
    FullTimestamp: true,
  })
  loger.SetLevel(logger.TraceLevel)
  hoom.SetLogger(loger)
	return
}

func main(){
	target := &net.TCPAddr{IP: localhost, Port: 12348}
	user := hoom.LogMember(0x02, "example-user")
	client, err := user.Dial(target)
	must(err)
	defer client.Close()
	loger.Infof("Client connected to %v", target)
	ping, err := client.Ping()
	must(err)
	loger.Infof("Ping: %v", ping)

	err = client.Join(roomid)
	must(err)
	loger.Infof("Joined room %d", roomid)

	listener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: localhost, Port: 12347})
	must(err)
	loger.Infof("Listening: %v", listener.Addr())
	var conn net.Conn
	for {
		conn, err = listener.Accept()
		must(err)
		go func(conn net.Conn){
			rwc, err := client.Dial(roomid)
			must(err)
			loger.Infof("client %v dialed\n", conn.RemoteAddr())
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
