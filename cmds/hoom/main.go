
package main

import (
	"bufio"
	"flag"
	"io"
	// "net"
	"os"
	"strings"
	"strconv"

	"github.com/kmcsr/go-hoom"
)

var (
	debug bool = false
	binary_mode bool = false
	username string = ""
	userid string = "" // do not use uint32, probably will change to uuid
	loginToken string = ""
)

func init(){
	flag.BoolVar(&debug, "debug", debug, "enable debug messages")
	// flag.BoolVar(&binary_mode, "binary", binary_mode, "use binary rpc mode")
	flag.StringVar(&username, "username", "", "client user name")
	flag.StringVar(&userid, "userid", "", "client user id")
	flag.StringVar(&loginToken, "token", "", "login token")
	flag.Parse()
	// TODO: log in user
	uid, err := strconv.ParseUint(userid, 10, 32)
	if err != nil {
		panic(err)
	}
	loggedUser = hoom.NewMember((uint32)(uid), username)
}

func main(){
	commander := TextCommander{}
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		cmd := scanner.Text()
		fields := strings.Fields(cmd)
		if len(fields) == 0 {
			continue
		}
		cmd, fields = fields[0], fields[1:]
		res, err := commander.Execute(cmd, fields...)
		if err != nil {
			panic(err)
		}
		if _, err = io.WriteString(os.Stdout, res); err != nil {
			panic(err)
		}
	}
}
