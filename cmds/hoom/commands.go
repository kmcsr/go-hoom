
package main

import (
	"errors"
	"fmt"
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

	// Server Side
	ServeCmd struct{}
	CreateRoomCmd struct{}
	ListCmd struct{}

	// Client Side
	JoinCmd struct{}
	QueryCmd struct{}
)

func initCommands()(cmds map[string]Command){
	cmds = make(map[string]Command)
	cmds["echo"] = EchoCmd{}

	cmds["serve"] = ServeCmd{}
	cmds["create"] = CreateRoomCmd{}
	cmds["list"] = ListCmd{}

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
	if len(fields) > 1 {
		return nil, FieldsNumWrong
	}
	var tcpaddr *net.TCPAddr = nil
	if len(fields) >= 1 {
		addr := fields[0]
		tcpaddr, err = net.ResolveTCPAddr("tcp", addr)
		if err != nil {
			return
		}
	}
	if hoomServer != nil {
		return nil, errors.New("Server already started")
	}
	server := loggedUser.NewServer(tcpaddr)
	if err = server.Listen(); err != nil {
		return
	}
	resn = append(resn, server.ListenAddr().String())
	hoomServer = server
	go func(){
		if err := server.Serve(); err != nil {
			loger.Panic(err)
		}
	}()
	return
}

func (CreateRoomCmd)Execute(fields ...string)(resn []interface{}, err error){
	if len(fields) != 2 {
		return nil, FieldsNumWrong
	}
	tgadr := fields[0]
	name := fields[1]
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

func (ListCmd)Execute(fields ...string)(resn []interface{}, err error){
	if len(fields) != 1 {
		return nil, FieldsNumWrong
	}
	roomid, err := strconv.ParseUint(fields[0], 10, 32)
	if err != nil {
		return
	}
	if hoomServer == nil {
		return nil, errors.New("Server not created")
	}
	room := hoomServer.GetRoom((uint32)(roomid))
	if room == nil {
		return nil, fmt.Errorf("Room(%d) not exists", roomid)
	}
	for _, m := range room.Members() {
		resn = append(resn, fmt.Sprintf("%d:%s", m.Id(), m.Name()))
	}
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
		if client, err = loggedUser.Dial(tcpaddr); err != nil {
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
	if _, err = client.Join((uint32)(roomid)); err != nil {
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
				loger.Panic(err)
				return
			}
			loger.Tracef("Accept conn %v", conn.RemoteAddr())
			go func(conn net.Conn){
				rwc, err := client.Dial((uint32)(roomid))
				if err != nil {
					conn.Close()
					loger.Errorf("Cannot dial room %d: %v", roomid, err.Error())
					return
				}
				done := ioProxy(rwc, conn)
				go func(){
					select {
					case <-done:
						loger.Tracef("Conn %v was done", conn.RemoteAddr())
					}
				}()
			}(conn)
		}
	}()
	return
}
