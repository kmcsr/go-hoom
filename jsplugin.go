
package hoom

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/kmcsr/goja"
	"github.com/kmcsr/goja/extern/require"
	"github.com/kmcsr/goja/extern/process"
	"github.com/kmcsr/goja/extern/console"
	"github.com/kmcsr/goja/extern/socket"
)

type Map = map[string]any

type PluginData struct{
	Room *Room
	ServeAddr net.Addr
}

var pluginsDir = initPluginsDir()

func initPluginsDir()(dir string){
	var err error
	if dir, err = os.UserCacheDir(); err != nil {
		dir = ".cache"
	}
	dir = filepath.Join(dir, "go_hoom", "plugins")
	return
}

func GetPluginSrc(typ uuid.UUID)(src string, err error){
	src = filepath.Join(pluginsDir, "minecraft.js")
	if _, err := os.Stat(src); os.IsNotExist(err) {
		src = filepath.Join("plugins", "minecraft.js")
	}
	return
}

type JsPlugin struct{
	running bool
	onload func(data PluginData)(err error)
	onunload func()(err error)
}

func (p *JsPlugin)Load(data PluginData)(err error){
	if p.running {
		return
	}
	p.running = true
	return p.onload(data)
}

func (p *JsPlugin)Unload()(err error){
	if !p.running {
		return
	}
	p.running = false
	return p.onunload()
}

func LoadPluginWithCtx(ctx context.Context, src string, vm *goja.Runtime)(plugin *JsPlugin, err error){
	program, err := compileJsFile(src)
	if err != nil {
		return
	}
	done := make(chan struct{})
	vm.RunOnLoop(func(*goja.Runtime)(error){
		defer close(done)

		var pluv goja.Value
		if pluv, err = vm.RunProgram(program); err != nil {
			return nil
		}
		plum := pluv.Export().(Map)
		onload, err := getMethod(vm, plum, "onload")
		if err != nil {
			return nil
		}
		onunload, err := getMethod(vm, plum, "onunload")
		if err != nil {
			return nil
		}
		plugin = new(JsPlugin)
		plugin.onload = func(data PluginData)(err error){
			done := make(chan struct{})
			vm.RunOnLoop(func(*goja.Runtime)(error){
				defer close(done)
				_, err = onload(pluv, vm.ToValue(Map{
					"id": data.Room.Id(),
					"owner": member2map(data.Room.Owner()),
					"typeId": data.Room.TypeId().String(),
					"name": data.Room.Name(),
					"desc": data.Room.Desc(),
					"addr": data.ServeAddr.String(),
				}))
				return nil
			})
			<-done
			return
		}
		plugin.onunload = func()(err error){
			done := make(chan struct{})
			vm.RunOnLoop(func(*goja.Runtime)(error){
				defer close(done)
				_, err = onunload(pluv)
				return nil
			})
			<-done
			return
		}
		return nil
	})
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-done:
	}
	return
}

func LoadPlugin(src string, vm *goja.Runtime)(plugin *JsPlugin, err error){
	return LoadPluginWithCtx(context.Background(), src, vm)
}

func compileJsFile(src string)(prog *goja.Program, err error){
	var data []byte
	if data, err = os.ReadFile(src); err != nil {
		return
	}
	return goja.Compile(src, (string)(data), true)
}

func NewJsRuntime()(vm *goja.Runtime){
	vm = goja.New()
	vm.SetFieldNameMapper(goja.UncapFieldNameMapper())

	registry := require.NewRegistry()
	console.RegisterWithLogger(registry, loger)
	socket.RegisterNativeModule(registry)

	registry.Enable(vm)
	console.Enable(vm)
	process.Enable(vm)
	return
}

func getMethod(runtime *goja.Runtime, mp Map, name string)(callable goja.Callable, err error){
	callv, ok := mp[name]
	if !ok {
		return nil, fmt.Errorf("Plugin missing method '%s'", err)
	}
	callable, ok = goja.AssertFunction(runtime.ToValue(callv))
	if !ok {
		return nil, fmt.Errorf("Plugin property '%s' isn't a callable", err)
	}
	return
}

func member2map(m *Member)(Map){
	return Map{
		"id": m.Id(),
		"name": m.Name(),
	}
}
