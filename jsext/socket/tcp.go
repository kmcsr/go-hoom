
package js_socket

import (
	"net"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/eventloop"
	events "github.com/kmcsr/go-hoom/jsext/events"
)

type TCPConn struct{
	*events.EventEmitter

	conn *net.TCPConn
	runtime *goja.Runtime

	Local goja.Value
	Remote goja.Value
}

func wrapTCPConn(loop *eventloop.EventLoop, runtime *goja.Runtime, conn *net.TCPConn)(v goja.Value){
	c := &TCPConn{
		EventEmitter: events.NewEventEmitter(loop, runtime),
		conn: conn,
		runtime: runtime,
		Local: runtime.ToValue(conn.LocalAddr().String()),
		Remote: runtime.ToValue(conn.RemoteAddr().String()),
	}
	v = runtime.ToValue(c)
	go func(){
		defer c.EmitAsync("close")
		var (
			buf = make([]byte, 1024 * 32)
		)
		for {
			n, err := conn.Read(buf)
			if err != nil {
				c.EmitAsync("error", c.runtime.ToValue(err))
				return
			}
			c.EmitAsync("data", c.runtime.ToValue(buf[:n]))
		}
	}()
	return
}

func (c *TCPConn)Close(){
	if err := c.conn.Close(); err != nil {
		panic(c.runtime.NewGoError(err))
	}
}

func (c *TCPConn)Send(buf []byte){
	if _, err := c.conn.Write(buf); err != nil {
		panic(c.runtime.NewGoError(err))
	}
}

type TCPListener struct{
	*events.EventEmitter

	listener *net.TCPListener
	runtime *goja.Runtime

	Addr goja.Value
}

func wrapTCPListener(loop *eventloop.EventLoop, runtime *goja.Runtime, listener *net.TCPListener)(v goja.Value){
	l := &TCPListener{
		EventEmitter: events.NewEventEmitter(loop, runtime),
		listener: listener,
		runtime: runtime,
		Addr: runtime.ToValue(listener.Addr().String()),
	}
	v = runtime.ToValue(l)
	go func(){
		for {
			conn, err := l.listener.AcceptTCP()
			if err != nil {
				return
			}
			c := wrapTCPConn(loop, l.runtime, conn)
			l.EmitAsync("accept", c)
		}
	}()
	return
}

func (l *TCPListener)Close(a interface{}){
	if err := l.listener.Close(); err != nil {
		panic(l.runtime.NewGoError(err))
	}
}
