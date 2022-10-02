
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

	onunload func()
}

func newJoinedRoom(room *Room, token *RoomToken)(r *joinedRoom, err error){
	r = &joinedRoom{
		Room: room,
		token: token,
	}
	return
}

type Client struct{
	mem *AuthedMember
	conn *pio.Conn
	server *net.TCPAddr
	rooms map[uint32]*joinedRoom
}

func (m *AuthedMember)Dial(server *net.TCPAddr)(c *Client, err error){
	var conn *net.TCPConn
	conn, err = net.DialTCP("tcp", nil, server)
	if err != nil {
		return
	}
	c = &Client{
		mem: m,
		conn: pio.NewConn(conn, conn),
		server: server,
		rooms: make(map[uint32]*joinedRoom),
	}
	c.jnitPacket()
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

func (c *Client)jnitPacket(){
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
	var room *joinedRoom
	switch rs := res.(type) {
	case *SjoinPkt:
		if room, err = newJoinedRoom(rs.Room, rs.Token); err != nil {
			return
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
	con, err = net.DialTCP("tcp", nil, c.server)
	if err != nil {
		return
	}
	conn = pio.NewConn(con, con)
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
		err = fmt.Errorf("You didn't join the room(%d) yet", id)
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
