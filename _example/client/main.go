
package main

import (
	"encoding/base64"
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
	buf, err := os.ReadFile("token.txt")
	must(err)
	buf, err = base64.RawURLEncoding.DecodeString((string)(buf))
	must(err)
	token, err := hoom.ParseConnToken(buf)
	must(err)
	user, err := hoom.LogMember(0x02, "<token>")
	must(err)
	client, err := user.Dial(hoom.DialToken(token).
		AddHandshaker(hoom.UnsafeHandshaker).
		AddHandshaker(&hoom.RsaHandshaker{}))
	must(err)
	defer client.Close()
	loger.Infof("Client connected to %v", token.Target)
	ping, err := client.Ping()
	must(err)
	loger.Infof("Ping: %v", ping)

	room, err := client.Join(roomid)
	must(err)
	loger.Infof("Joined room %v", room)

	listener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: localhost, Port: 12347})
	must(err)
	loger.Infof("Listening: %v", listener.Addr())
	err = client.ServeRoom(roomid, listener)
	must(err)
}

func must(err error){
	if err != nil {
		panic(err)
	}
}
