
package main

import (
	"net/netip"
  "os"

  "github.com/sirupsen/logrus"
  "github.com/kmcsr/go-logger"
  logrusl "github.com/kmcsr/go-logger/logrus"
	"github.com/kmcsr/go-hoom"
)

var loger = initLogger()

func initLogger()(loger logger.Logger){
  loger = logrusl.New()
  loger.SetOutput(os.Stderr)
  logrusl.Unwrap(loger).SetFormatter(&logrus.TextFormatter{
    TimestampFormat: "2006-01-02 15:04:05.000",
    FullTimestamp: true,
  })
  loger.SetLevel(logger.InfoLevel)
  hoom.SetLogger(loger)
  return
}

var (
	loggedUser *hoom.AuthedMember
	hoomServer *hoom.Server = nil
	hoomClients = make(map[netip.AddrPort]*hoom.Client)
)

var cliUsage = `
hoom --userid <userid> --username <username> --token <login token>
`

var cliCommandsUsage = `
  echo [args...]
    echo arguments.
    :returns: the inputed arguments

  Server Side Commands
  ========
  serve [<addr>]
    create a server, listen and serve at <addr> or a auto selected address.
    :args:addr: the listening address, allow auto select
    :returns: the listening address

  create <target> <name>
    create a room named <name> that will connect to <target>.
    :args:target: the target adddress
    :args:name: the room's public name
    :returns: the room id

  list <roomid>
    list room's members
    :args:roomid: the room's id
    :returns: members. format=<id>:<name>

  Client Side Commands
  ========
  join <addr> <roomid>
    connect to server at <addr> if not connected, and then join the room.
    :args:addr: the server's address
    :args:roomid: the room id to join on the server
    :returns: the listening address

  query <addr> <roomid>
    List the members of the room in <addr>
    :args:addr: the server's address
    :args:roomid: the room's id
    :returns: members. format=<id>:<name>
`
