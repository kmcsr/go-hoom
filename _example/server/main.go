
package main

import (
	crand "crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"net"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/kmcsr/go-logger"
	logrusl "github.com/kmcsr/go-logger/logrus"
	"github.com/kmcsr/go-hoom"
)

var localhost = net.IPv4(127, 0, 0, 1)
var server_addr = &net.TCPAddr{IP: localhost, Port: 12348}

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
	owner, err := hoom.NoAuthMemberServer.AuthMember("example_server", "")
	if err != nil {
		panic(err)
	}
	prikey, err := rsa.GenerateKey(crand.Reader, 2048)
	if err != nil {
		panic(err)
	}
	handshaker := &hoom.RsaHandshaker{
		Key: prikey,
	}
	token := handshaker.ConnToken(server_addr)
	err = os.WriteFile("token.txt", ([]byte)(base64.RawURLEncoding.EncodeToString(token.Encode())), 0666)
	if err != nil {
		panic(err)
	}
	server := owner.NewServer(server_addr).AddHandshaker(handshaker).AddHandshaker(hoom.UnsafeHandshaker)
	room := server.NewRoom("example-room", &net.TCPAddr{IP: localhost, Port: 25565})
	loger.Info("new room id:", room.Id())
	loger.Info("server.ListenAndServe")
	err = server.ListenAndServe()
	if err != nil {
		panic(err)
	}
}
