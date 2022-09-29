
package main

import (
	"io"
)

func ioProxy(a, b io.ReadWriteCloser)(done <-chan struct{}){
	done0 := make(chan struct{}, 0)
	go func(){
		defer close(done0)
		defer a.Close()
		defer b.Close()
		io.Copy(b, a)
	}()
	go func(){
		defer a.Close()
		defer b.Close()
		io.Copy(a, b)
	}()
	return done0
}
