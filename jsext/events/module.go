
package js_events

import(
	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	"github.com/dop251/goja_nodejs/eventloop"
)

const ModuleName = "node:events"

type Constructor = func(call goja.ConstructorCall, runtime *goja.Runtime)(v *goja.Object)

func newEventEmitter(loop *eventloop.EventLoop)(Constructor){
	return func(call goja.ConstructorCall, runtime *goja.Runtime)(v *goja.Object){
		emitter := NewEventEmitter(loop, runtime)
		return runtime.ToValue(emitter).ToObject(runtime)
	}
}

func Require(loop *eventloop.EventLoop)(require.ModuleLoader){
	return func(runtime *goja.Runtime, module *goja.Object){
		o := module.Get("exports").(*goja.Object)
		o.Set("EventEmitter", newEventEmitter(loop))
	}
}

func Register(r *require.Registry, loop *eventloop.EventLoop){
	r.RegisterNativeModule(ModuleName, Require(loop))
}
