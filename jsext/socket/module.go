
package js_socket

import (
	"net"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	"github.com/dop251/goja_nodejs/eventloop"
)

const ModuleName = "socket"

type Function = func(call goja.FunctionCall, runtime *goja.Runtime)(v goja.Value)

func socketListen(loop *eventloop.EventLoop)(Function){
	return func(call goja.FunctionCall, runtime *goja.Runtime)(v goja.Value){
		var (
			network string
			addr string
		)
		network = call.Arguments[0].Export().(string)
		if len(call.Arguments) >= 2 {
			addr = call.Arguments[1].Export().(string)
		}
		switch network {
		case "tcp", "tcp4", "tcp6":
			adr, err := net.ResolveTCPAddr(network, addr)
			if err != nil {
				panic(runtime.ToValue(err))
			}
			lis, err := net.ListenTCP(network, adr)
			if err != nil {
				panic(runtime.ToValue(err))
			}
			return wrapTCPListener(loop, runtime, lis)
		case "udp", "udp4", "udp6":
			adr, err := net.ResolveUDPAddr(network, addr)
			if err != nil {
				panic(runtime.ToValue(err))
			}
			conn, err := net.ListenUDP(network, adr)
			if err != nil {
				panic(runtime.ToValue(err))
			}
			return wrapUDPConn(loop, runtime, conn)
		default:
			panic(runtime.ToValue("Unknown network '" + network + "'"))
		}
	}
}

func socketDial(loop *eventloop.EventLoop)(Function){
	return func(call goja.FunctionCall, runtime *goja.Runtime)(v goja.Value){
		var (
			network string
			raddr string
			laddr string
		)
		network = call.Arguments[0].Export().(string)
		if len(call.Arguments) >= 2 {
			raddr = call.Arguments[1].Export().(string)
			if len(call.Arguments) >= 3 {
				laddr = call.Arguments[2].Export().(string)
			}
		}
		switch network {
		case "tcp", "tcp4", "tcp6":
			radr, err := net.ResolveTCPAddr(network, raddr)
			if err != nil {
				panic(runtime.ToValue(err))
			}
			ladr, err := net.ResolveTCPAddr(network, laddr)
			if err != nil {
				panic(runtime.ToValue(err))
			}
			conn, err := net.DialTCP(network, ladr, radr)
			if err != nil {
				panic(runtime.ToValue(err))
			}
			return wrapTCPConn(loop, runtime, conn)
		case "udp", "udp4", "udp6":
			radr, err := net.ResolveUDPAddr(network, raddr)
			if err != nil {
				panic(runtime.ToValue(err))
			}
			ladr, err := net.ResolveUDPAddr(network, laddr)
			if err != nil {
				panic(runtime.ToValue(err))
			}
			conn, err := net.DialUDP(network, ladr, radr)
			if err != nil {
				panic(runtime.ToValue(err))
			}
			return wrapUDPConn(loop, runtime, conn)
		default:
			panic(runtime.ToValue("Unknown network '" + network + "'"))
		}
	}
}

func Require(loop *eventloop.EventLoop)(require.ModuleLoader){
	return func(runtime *goja.Runtime, module *goja.Object){
		o := module.Get("exports").(*goja.Object)
		o.Set("listen", socketListen(loop))
		o.Set("dial", socketDial(loop))
	}
}

func Register(r *require.Registry, loop *eventloop.EventLoop){
	r.RegisterNativeModule(ModuleName, Require(loop))
}
