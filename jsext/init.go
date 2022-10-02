
package jsext

import (
	"github.com/dop251/goja_nodejs/require"
	"github.com/dop251/goja_nodejs/eventloop"
	events "github.com/kmcsr/go-hoom/jsext/events"
	socket "github.com/kmcsr/go-hoom/jsext/socket"
)

func Register(r *require.Registry, loop *eventloop.EventLoop){
	events.Register(r, loop)
	socket.Register(r, loop)
}
