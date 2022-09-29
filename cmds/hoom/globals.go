
package main

import (
	"net/netip"
	"github.com/kmcsr/go-hoom"
)

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

  serve [<addr>]
    create a server, listen and serve at <addr> or a auto selected address.
    :args:addr: the listening address, allow auto select
    :returns: the listening address

  create <name> <target>
    create a room named <name> that will connect to <target>.
    :args:name: the room's public name
    :args:target: the target adddress
    :returns: the room id

  join <addr> <roomid>
    connect to server at <addr> if not connected,
    and then joined the room that id is <roomid>.
    :args:addr: the server's listening address
    :args:roomid: the room id to join on the server
    :returns: the listening address
`
