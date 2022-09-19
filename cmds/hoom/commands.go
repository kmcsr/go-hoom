
package main

import (
	"errors"
	"net"
	"strconv"
)

var (
	FieldsNumWrong = errors.New("Command fields length wrong")
)

type Command interface{
	Execute(fields ...string)(resn []interface{}, err error)
}

var commands = initCommands()

type (
	EchoCmd struct{}
	ServeCmd struct{}
	CreateRoomCmd struct{}
	JoinCmd struct{}
)

func initCommands()(cmds map[string]Command){
	cmds = make(map[string]Command)
	cmds["echo"] = EchoCmd{}
	cmds["serve"] = ServeCmd{}
	cmds["create"] = CreateRoomCmd{}
	cmds["join"] = JoinCmd{}
	return
}

func (EchoCmd)Execute(fields ...string)(resn []interface{}, err error){
	resn = make([]interface{}, len(fields))
	for i, r := range fields {
		resn[i] = r
	}
	return
}

func (ServeCmd)Execute(fields ...string)(resn []interface{}, err error){
	if len(fields) != 1 {
		return nil, FieldsNumWrong
	}
	addr := fields[0]
	tcpaddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return
	}
	if hoomServer != nil {
		return nil, errors.New("Server already started")
	}
	server := loggedUser.NewServer(tcpaddr)
	hoomServer = server
	go func(){
		if err := server.ListenAndServe(); err != nil {
			panic(err)
		}
	}()
	return
}

func (CreateRoomCmd)Execute(fields ...string)(resn []interface{}, err error){
	if len(fields) != 2 {
		return nil, FieldsNumWrong
	}
	name := fields[0]
	tgadr := fields[1]
	target, err := net.ResolveTCPAddr("tcp", tgadr)
	if err != nil {
		return
	}
	if hoomServer == nil {
		return nil, errors.New("Server not created")
	}
	room := hoomServer.NewRoom(name, target)
	resn = append(resn, room.Id())
	return
}

func (JoinCmd)Execute(fields ...string)(resn []interface{}, err error){
	if len(fields) != 2 {
		return nil, FieldsNumWrong
	}
	addr := fields[0]
	roomid, err := strconv.ParseUint(fields[1], 10, 32)
	if err != nil {
		return
	}
	tcpaddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return
	}
	netap := tcpaddr.AddrPort()
	client, ok := hoomClients[netap]
	if !ok {
		if client, err = loggedUser.DialServer(tcpaddr); err != nil {
			return
		}
		hoomClients[netap] = client
		go func(){
			select {
			case <-client.Context().Done():
				delete(hoomClients, netap)
			}
		}()
	}
	if err = client.Join((uint32)(roomid)); err != nil {
		return
	}
	listener, err := net.ListenTCP("tcp", nil)
	if err != nil {
		return
	}
	go func(){
		select {
		case <-client.Context().Done():
			listener.Close()
		}
	}()
	resn = append(resn, listener.Addr().String())
	go func(){
		defer listener.Close()
		var (
			conn net.Conn
			err error
		)
		for {
			conn, err = listener.Accept()
			if err != nil {
				panic(err)
				return
			}
			_, _, err = client.Dial((uint32)(roomid), conn)
			if err != nil {
				println("error Cannot dial room:", roomid, ":", err.Error())
				return
			}
		}
	}()
	return
}
