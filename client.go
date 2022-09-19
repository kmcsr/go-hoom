
package hoom

import (
	"context"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/kmcsr/go-pio"
)


type connRoom struct{
	room uint32
	w io.Writer
}

type Client struct{
	mem *Member
	conn *pio.Conn
	rooms map[uint32]*Room
	conns map[uint32]connRoom
}

func (m *Member)DialServer(server *net.TCPAddr)(c *Client, err error){
	var conn *net.TCPConn
	conn, err = net.DialTCP("tcp", nil, server)
	if err != nil {
		return
	}
	c = &Client{
		mem: m,
		conn: pio.NewConn(conn, conn),
		rooms: make(map[uint32]*Room),
		conns: make(map[uint32]connRoom),
	}
	c.initConn()
	go c.conn.Serve()
	if err = c.conn.Send(&CbindPkt{
		Mem: m,
	}); err != nil {
		c.conn.Close()
		return nil, err
	}
	return
}

func (c *Client)Ping()(ping time.Duration, err error){
	return c.conn.Ping()
}

func (c *Client)Close()(error){
	return c.conn.Close()
}

func (c *Client)Context()(context.Context){
	return c.conn.Context()
}

func (c *Client)Rooms()(rooms []*Room){
	for _, r := range c.rooms {
		rooms = append(rooms, r)
	}
	return
}

func (c *Client)GetRoom(id uint32)(r *Room){
	return c.rooms[id]
}

func (c *Client)Disconnect(id uint32)(err error){
	if _, ok := c.pop(id); ok {
		if err = c.conn.Send(&CclosePkt{
			ConnId: id,
		}); err != nil {
			return
		}
	}
	return
}

func (c *Client)pop(id uint32)(rc connRoom, ok bool){
	if rc, ok = c.conns[id]; ok {
		delete(c.conns, id)
	}
	return
}

func (c *Client)initConn(){
	c.conn.AddPacket(func()(pio.PacketBase){ return &SjoinPkt  {c: c} })
	c.conn.AddPacket(func()(pio.PacketBase){ return &SjoinBPkt {c: c} })
	c.conn.AddPacket(func()(pio.PacketBase){ return &SleavePkt {c: c} })
	c.conn.AddPacket(func()(pio.PacketBase){ return &SleaveBPkt{c: c} })
	c.conn.AddPacket(func()(pio.PacketBase){ return &SerrorPkt {c: c} })
	c.conn.AddPacket(func()(pio.PacketBase){ return &SdialPkt  {c: c} })
	c.conn.AddPacket(func()(pio.PacketBase){ return &SclosePkt {c: c} })
	c.conn.AddPacket(func()(pio.PacketBase){ return &SsendPkt  {c: c} })
}

func (c *Client)Join(id uint32)(err error){
	var res pio.PacketBase
	if res, err = c.conn.Ask(&CjoinPkt{
		RoomId: id,
	}); err != nil {
		return
	}
	var room *Room
	switch rs := res.(type) {
	case *SjoinPkt:
		room = rs.Room
	case *SerrorPkt:
		return fmt.Errorf("Dial error: %s", rs.Error)
	default:
		panic("Unexpected result")
	}
	c.rooms[id] = room
	return
}

func (c *Client)Dial(id uint32, rw io.ReadWriter)(ses uint32, done <-chan error, err error){
	var res pio.PacketBase
	if res, err = c.conn.Ask(&CdialPkt{
		RoomId: id,
	}); err != nil {
		return
	}
	switch rs := res.(type) {
	case *SdialPkt:
		ses = rs.ConnId
	case *SerrorPkt:
		err = fmt.Errorf("Dial error: %s", rs.Error)
		return
	default:
		panic("Unexpected result")
	}
	c.conns[ses] = connRoom{id, rw}

	ch := make(chan error, 1)
	done = ch
	go func(){
		defer c.Disconnect(ses)
		defer close(ch)
		var (
			buf = make([]byte, 1024 * 128) // 128 KB
			n int
			er error
		)
		for {
			if n, er = rw.Read(buf); er != nil {
				if er == io.EOF {
					er = nil
				}
				break
			}
			if er = c.conn.Send(&CsendPkt{
				ConnId: ses,
				Data: buf[:n],
			}); er != nil {
				break
			}
		}
		if er != nil {
			ch <- er
		}
	}()
	return
}
