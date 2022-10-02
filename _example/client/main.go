
package main

import (
	"errors"
	"io"
	"net"
	"os"

  "github.com/sirupsen/logrus"
  "github.com/kmcsr/go-logger"
  logrusl "github.com/kmcsr/go-logger/logrus"
	"github.com/kmcsr/go-hoom"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	"github.com/dop251/goja_nodejs/eventloop"
	"github.com/dop251/goja_nodejs/process"
	"github.com/kmcsr/go-hoom/jsext"
	console "github.com/kmcsr/go-hoom/jsext/console"
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
	user, err := hoom.LogMember(0x02, "<token>")
	must(err)
	client, err := user.Dial(target)
	must(err)
	defer client.Close()
	loger.Infof("Client connected to %v", target)
	ping, err := client.Ping()
	must(err)
	loger.Infof("Ping: %v", ping)

	room, err := client.Join(roomid)
	must(err)
	loger.Infof("Joined room %d", roomid)

	var (
		onload func(addr net.Addr)(err error)
		onunload func()
	)
	{
		program, err := compileJsFile("plugins/minecraft.js")
		if err != nil {
			return
		}
		registry := require.NewRegistry()
		console.RegisterWithLogger(registry, loger)
		loop := eventloop.NewEventLoop(eventloop.WithRegistry(registry))
		jsext.Register(registry, loop)
		loop.Start()
		defer loop.Stop()
		done := make(chan struct{})
		loop.RunOnLoop(func(vm *goja.Runtime){
			defer close(done)
			vm.SetFieldNameMapper(goja.UncapFieldNameMapper())
			process.Enable(vm)

			v, err := vm.RunProgram(program)
			if err != nil {
				panic(err)
				return
			}
			plugin := v.Export().(map[string]interface{})
			onload_, ok := goja.AssertFunction(vm.ToValue(plugin["on_load"]))
			if !ok {
				panic(errors.New("plugin missing method 'on_load'"))
				return
			}
			onunload_, ok := goja.AssertFunction(vm.ToValue(plugin["on_unload"]))
			if !ok {
				panic(errors.New("plugin missing method 'on_unload'"))
				return
			}
			onload = func(addr net.Addr)(err error){
				done := make(chan struct{})
				loop.RunOnLoop(func(vm *goja.Runtime){
					defer close(done)
					_, err = onload_(v, vm.ToValue(Map{
						"id": room.Id(),
						"owner": member2map(room.Owner()),
						"typeId": room.TypeId().String(),
						"name": room.Name(),
						"desc": room.Desc(),
						"addr": addr.String(),
					}))
				})
				<-done
				return
			}
			onunload = func(){
				done := make(chan struct{})
				loop.RunOnLoop(func(vm *goja.Runtime){
					defer close(done)
					onunload_(v)
				})
				<-done
			}
		})
		select {
		case <-done:
		}
	}

	listener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: localhost, Port: 12347})
	must(err)
	loger.Infof("Listening: %v", listener.Addr())
	err = onload(listener.Addr())
	must(err)
	defer onunload()
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

type Map = map[string]any

func member2map(m *hoom.Member)(Map){
	return Map{
		"id": m.Id(),
		"name": m.Name(),
	}
}

func compileJsFile(src string)(prog *goja.Program, err error){
	var data []byte
	if data, err = os.ReadFile(src); err != nil {
		return
	}
	return goja.Compile(src, (string)(data), true)
}
