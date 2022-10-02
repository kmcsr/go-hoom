
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	// "net"
	"os"
	"strings"
	"strconv"

  "github.com/kmcsr/go-logger"
	"github.com/kmcsr/go-hoom"
)

var (
	debug bool = false
	trace bool = false
	binary_mode bool = false
	userid string = "" // do not use uint32, probably will change to uuid
	loginToken string = ""
)

func init(){
	flag.BoolVar(&debug, "debug", debug, "enable debug messages")
	flag.BoolVar(&trace, "trace", trace, "enable trace messages")
	// flag.BoolVar(&binary_mode, "binary", binary_mode, "use binary rpc mode")
	flag.StringVar(&userid, "userid", "", "client user id")
	flag.StringVar(&loginToken, "token", "", "login token")
	flag.Usage = func(){
		out := flag.CommandLine.Output()
		fmt.Fprintln(out, "Usage of Hoom-cli:")
		fmt.Fprintln(out, cliUsage)
		fmt.Fprintln(out, "Args:")
		fmt.Fprintln(out)
		flag.CommandLine.PrintDefaults()
		fmt.Fprintln(out, "\nCommands:")
		fmt.Fprint(out, cliCommandsUsage)
	}

	flag.Parse()
	if len(userid) == 0 || len(loginToken) == 0 {
		flag.Usage()
		os.Exit(2)
	}

	if trace {
		debug = true
	}
	if debug {
		loger.SetLevel(logger.DebugLevel)
	}
	if trace {
		loger.SetLevel(logger.TraceLevel)
	}

	// TODO: log in user
	uid, err := strconv.ParseUint(userid, 10, 32)
	if err != nil {
		loger.Panicf("Cannot parse userid: %v", err.Error())
	}
	loggedUser, err = hoom.LogMember((uint32)(uid), loginToken)
	if err != nil {
		loger.Panicf("Cannot login user: %v", err.Error())
	}
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
			loger.Panic(err)
		}
		if _, err = io.WriteString(os.Stdout, res); err != nil {
			loger.Panic(err)
		}
	}
}
