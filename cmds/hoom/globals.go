
package main

import (
	"net/netip"
	"github.com/kmcsr/go-hoom"
)

var (
	loggedUser *hoom.Member
	hoomServer *hoom.Server = nil
	hoomClients = make(map[netip.AddrPort]*hoom.Client)
)
