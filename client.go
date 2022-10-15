
package hoom

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/kmcsr/go-pio/encoding"
	"github.com/kmcsr/go-pio"
)

type joinedRoom struct{
	*Room
	token *RoomToken
}

type Client struct{
	mem *AuthedMember
	conn *pio.Conn
	config *DialConfig
	rooms map[uint32]joinedRoom
}

func (m *AuthedMember)Dial(config *DialConfig)(c *Client, err error){
	var conn *net.TCPConn
	conn, err = net.DialTCP("tcp", nil, config.Target.(*net.TCPAddr))
	if err != nil {
		return
	}
	var rw io.ReadWriteCloser
	if rw, err = m.handshake(conn, config); err != nil {
		conn.Close()
		return
	}
	c = &Client{
		mem: m,
		conn: pio.NewConn(rw, rw),
		config: config,
		rooms: make(map[uint32]joinedRoom),
	}
	c.initPacket()
	c.conn.OnPktNotFound = func(id uint32, body encoding.Reader){
		loger.Warn("hoom.Client: Unexpected packet id:", id)
	}
	go c.conn.Serve()
	var res pio.PacketBase
	if res, err = c.conn.Ask(&CbindPkt{
		Mem: m.GetMem(),
	}); err != nil {
		c.conn.Close()
		return nil, err
	}
	switch rs := res.(type) {
	case pio.Ok:
	case *SerrorPkt:
		return nil, fmt.Errorf("Cannot bind member: %s", rs.Error)
	default:
		panic("Unexpected result")
	}
	return
}

func (c *Client)Ping()(ping time.Duration, err error){
	return c.conn.Ping()
}

func (c *Client)Close()(err error){
	return c.conn.Close()
}

func (c *Client)Context()(context.Context){
	return c.conn.Context()
}

func (c *Client)Rooms()(rooms []*Room){
	rooms = make([]*Room, 0, len(c.rooms))
	for _, r := range c.rooms {
		rooms = append(rooms, r.Room)
	}
	return
}

func (c *Client)GetRoom(id uint32)(r *Room){
	if r, ok := c.rooms[id]; ok {
		return r.Room
	}
	return
}

func (c *Client)initPacket(){
	c.conn.AddPacket(func()(pio.PacketBase){ return &SjoinPkt  {c: c} })
	c.conn.AddPacket(func()(pio.PacketBase){ return &SjoinBPkt {c: c} })
	c.conn.AddPacket(func()(pio.PacketBase){ return &SleavePkt {c: c} })
	c.conn.AddPacket(func()(pio.PacketBase){ return &SleaveBPkt{c: c} })
	c.conn.AddPacket(func()(pio.PacketBase){ return &SerrorPkt {} })
}

func (c *Client)Join(id uint32)(rm *Room, err error){
	var res pio.PacketBase
	if res, err = c.conn.Ask(&CjoinPkt{
		RoomId: id,
	}); err != nil {
		return
	}
	var room joinedRoom
	switch rs := res.(type) {
	case *SjoinPkt:
		room = joinedRoom{
			Room: rs.Room,
			token: rs.Token,
		}
	case *SerrorPkt:
		return nil, fmt.Errorf("Join error: %s", rs.Error)
	default:
		panic("Unexpected result")
	}
	c.rooms[id] = room
	rm = room.Room
	return
}

func (c *Client)dial()(conn *pio.Conn, err error){
	var con *net.TCPConn
	con, err = net.DialTCP("tcp", nil, c.config.Target.(*net.TCPAddr))
	if err != nil {
		return
	}
	var rw io.ReadWriteCloser
	if rw, err = c.mem.handshake(con, c.config); err != nil {
		con.Close()
		return
	}
	conn = pio.NewConn(rw, rw)
	conn.AddPacket(func()(pio.PacketBase){ return &SerrorPkt {} })
	conn.OnPktNotFound = func(id uint32, body encoding.Reader){
		loger.Warn("Unexpected packet id:", id)
	}
	go conn.Serve()
	return
}

func (c *Client)Dial(id uint32)(conn io.ReadWriteCloser, err error){
	room, ok := c.rooms[id]
	if !ok {
		err = fmt.Errorf("Room(%d) wasn't connected", id)
		return
	}
	var con *pio.Conn
	if con, err = c.dial(); err != nil {
		return
	}
	defer func(){ if con != nil {
		con.Close()
	}}()
	var res pio.PacketBase
	if res, err = con.Ask(&CdialPkt{
		MemId: c.mem.Id(),
		Token: room.token,
	}); err != nil && !errors.Is(err, context.Canceled) {
		return
	}
	if err == nil {
		switch rs := res.(type) {
		case pio.Ok:
		case *SerrorPkt:
			err = fmt.Errorf("Dial error: %s", rs.Error)
			return
		default:
			panic("Unexpected result")
		}
	}
	loger.Trace("hoom.Client: pio.Conn streaming")
	if conn, err = con.AsStream(); err != nil {
		return
	}
	con = nil
	return
}

func (c *Client)ServeRoom(id uint32, listener net.Listener)(err error){
	room, ok := c.rooms[id]
	if !ok {
		err = fmt.Errorf("Room(%d) wasn't connected", id)
		return
	}
	defer listener.Close()
	loop := NewJsRuntime()
	loop.Start()
	defer loop.Stop()
	plusrc, err := GetPluginSrc(room.TypeId())
	if err != nil {
		return
	}
	plugin, err := LoadPlugin(plusrc, loop)
	if err != nil {
		return
	}
	if err = plugin.Load(PluginData{
		Room: room.Room,
		ServeAddr: listener.Addr(),
	}); err != nil {
		return
	}
	defer plugin.Unload()
	var (
		conn net.Conn
		rwc io.ReadWriteCloser
	)
	for {
		if conn, err = listener.Accept(); err != nil {
			return
		}
		loger.Tracef("hoom.Client: accept %v for room %d\n", conn.RemoteAddr(), id)
		if rwc, err = c.Dial(id); err != nil {
			return
		}
		loger.Tracef("hoom.Client: %v dialed to room %d\n", conn.RemoteAddr(), id)
		go func(conn net.Conn, rwc io.ReadWriteCloser){
			select {
			case <-ioProxy(conn, rwc):
				// DONE
			}
		}(conn, rwc)
	}
}
