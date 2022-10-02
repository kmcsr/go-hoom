
package main

import (
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
	owner, err := hoom.LogMember(0x01, "<token>")
  if err != nil {
    panic(err)
  }
	server := owner.NewServer(server_addr)
	room := server.NewRoom("example-room", &net.TCPAddr{IP: localhost, Port: 25565})
	loger.Info("new room id:", room.Id())
	loger.Info("server.ListenAndServe")
	server.ListenAndServe()
}
