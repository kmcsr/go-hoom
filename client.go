
package hoom

import (
	"context"
	"errors"
	"io"
	"net"

	"github.com/kmcsr/go-pio/encoding"
)

var (
	RoomNotExist = errors.New("Room not exist")
	RepeatConn = errors.New("Repeat connect to a same room")
)

type connRoom struct{
	*Room
	conn *net.TCPConn
}

type Client struct{
	m *Member
	rooms map[uint32]connRoom
}

func (c *Client)ConnTo(addr *net.TCPAddr)(room *Room, err error){
	var conn *net.TCPConn
	conn, err = net.DialTCP("tcp", nil, addr)
	if err != nil {
		return
	}
	defer func(){
		if conn != nil {
			conn.Close()
		}
	}()
	if err = c.m.WriteTo(encoding.WrapWriter(conn)); err != nil {
		return
	}
	room = new(Room)
	if err = room.ParseFrom(encoding.WrapReader(conn)); err != nil {
		return
	}
	if _, ok := c.rooms[room.Id()]; ok {
		return nil, RepeatConn
	}
	c.rooms[room.Id()] = connRoom{room, conn}
	conn = nil
	return
}

func (c *Client)GetRoom(id uint32)(r *Room, ok bool){
	var rc connRoom
	if rc, ok = c.rooms[id]; ok {
		r = rc.Room
	}
	return
}

func (c *Client)Disconnect(id uint32)(r *Room, err error){
	if rc, ok := c.pop(id); ok {
		r = rc.Room
		err = rc.conn.Close()
	}
	return
}

func (c *Client)pop(id uint32)(r connRoom, ok bool){
	r, ok = c.rooms[id]
	if ok {
		delete(c.rooms, id)
	}
	return
}

func (c *Client)Proxy(id uint32, rw io.ReadWriter)(err error){
	return c.ProxyWith(context.Background(), id, rw)
}

func (c *Client)ProxyWith(ctx context.Context, id uint32, rw io.ReadWriter)(err error){
	rc, ok := c.rooms[id]
	if !ok {
		return RoomNotExist
	}
	ch := make(chan struct{})
	go func(){
		defer close(ch)
		if _, err0 := io.Copy(rw, rc.conn); err0 != nil {
			err = err0
		}
	}()
	go func(){
		defer close(ch)
		if _, err0 := io.Copy(rc.conn, rw); err0 != nil {
			err = err0
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

