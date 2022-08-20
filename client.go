
package hoom

import (
	"context"
	"fmt"
	"io"
	"net"
	"time"

	// "github.com/kmcsr/go-pio/encoding"
	"github.com/kmcsr/go-pio"
)


type connRoom struct{
	*Room
	w io.Writer
}

type Client struct{
	mem *Member
	conn *pio.Conn
	rooms map[uint32]connRoom
}

func NewClient(m *Member, addr *net.TCPAddr)(c *Client, err error){
	var conn *net.TCPConn
	conn, err = net.DialTCP("tcp", nil, addr)
	if err != nil {
		return
	}
	c = &Client{
		mem: m,
		conn: pio.NewConn(conn, conn),
	}
	go c.serve()
	if err = c.conn.Send(&CbindPkt{
		Mem: m,
	}); err != nil {
		return
	}
	return
}

func (c *Client)Ping()(ping time.Duration, err error){
	return c.conn.Ping()
}

func (c *Client)GetRoom(id uint32)(r *Room){
	if rc, ok := c.rooms[id]; ok {
		r = rc.Room
	}
	return
}

func (c *Client)Disconnect(id uint32)(err error){
	if _, ok := c.pop(id); ok {
		if err = c.conn.Send(&CclosePkt{
			RoomId: id,
		}); err != nil {
			return
		}
	}
	return
}

func (c *Client)pop(id uint32)(rc connRoom, ok bool){
	if rc, ok = c.rooms[id]; ok {
		delete(c.rooms, id)
	}
	return
}

func (c *Client)initConn(){
	c.conn.AddPacket(func()(pio.PacketBase){ return &SjoinPkt  {c: c} })
	c.conn.AddPacket(func()(pio.PacketBase){ return &SjoinBPkt {c: c} })
	c.conn.AddPacket(func()(pio.PacketBase){ return &SleavePkt {c: c} })
	c.conn.AddPacket(func()(pio.PacketBase){ return &SleaveBPkt{c: c} })
	c.conn.AddPacket(func()(pio.PacketBase){ return &SerrorPkt {c: c} })
	c.conn.AddPacket(func()(pio.PacketBase){ return &SclosePkt {c: c} })
	c.conn.AddPacket(func()(pio.PacketBase){ return &SsendPkt  {c: c} })
}

func (c *Client)serve()(err error){
	c.initConn()
	return c.conn.Serve()
}

func (c *Client)Dial(id uint32, rw io.ReadWriter)(err error){
	return c.DialWith(context.Background(), id, rw)
}

func (c *Client)DialWith(ctx context.Context, id uint32, rw io.ReadWriter)(err error){
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
		return fmt.Errorf(rs.Error)
	default:
		panic("Unexpected result")
	}
	c.rooms[id] = connRoom{room, rw}

	ch := make(chan struct{})
	go func(){
		defer close(ch)
		var (
			buf = make([]byte, 1024 * 64)
			n int
			er error
		)
		for {
			if n, er = rw.Read(buf); er != nil {
				break
			}
			if er = c.conn.Send(&CsendPkt{
				RoomId: room.Id(),
				Data: buf[:n],
			}); er != nil {
				break
			}
		}
		if er != nil {
			err = er
		}
	}()
	select {
	case <-ch:
	case <-ctx.Done():
		close(ch)
		err = ctx.Err()
	}
	return
}

