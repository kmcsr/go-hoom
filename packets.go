
package hoom

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/kmcsr/go-pio/encoding"
	"github.com/kmcsr/go-pio"
)

type (
	CbindPkt struct{
		Mem *Member

		authServer MemberServer
		server *Server
		conn *pio.Conn
		alive context.CancelFunc
	}
	CjoinPkt struct{
		RoomId uint32

		s *serverConn
	}
	CleavePkt struct{
		RoomId uint32

		s *serverConn
	}

	CdialPkt struct{
		MemId string
		Token *RoomToken

		server *Server
		conn *pio.Conn
		alive context.CancelFunc
	}
)

type (
	SjoinPkt struct{
		Room *Room
		Token *RoomToken

		authServer MemberServer
		c *Client
	}
	SjoinBPkt struct{
		RoomId uint32
		Mem *Member

		authServer MemberServer
		c *Client
	}
	SleavePkt struct{
		RoomId uint32
		Reason string

		c *Client
	}
	SleaveBPkt struct{
		RoomId uint32
		MemId string

		c *Client
	}
	SerrorPkt struct{
		Error string
	}
)


var (
	_ pio.PacketAsk = (*CbindPkt)(nil)
	_ pio.PacketAsk = (*CjoinPkt)(nil)
	_ pio.PacketAsk = (*CleavePkt)(nil)
	_ pio.PacketAsk = (*CdialPkt)(nil)
)

func (p *CbindPkt) PktId()(uint32){ return 0x81 }
func (p *CjoinPkt) PktId()(uint32){ return 0x82 }
func (p *CleavePkt)PktId()(uint32){ return 0x83 }
func (p *CdialPkt) PktId()(uint32){ return 0x88 }

func (p *CbindPkt)WriteTo(w encoding.Writer)(err error){
	if err = p.authServer.WriteMember(w, p.Mem); err != nil {
		return
	}
	return
}

func (p *CbindPkt)ParseFrom(r encoding.Reader)(err error){
	if p.Mem, err = p.authServer.ParseMember(r); err != nil {
		return
	}
	return
}

func (p *CbindPkt)Ask()(res pio.PacketBase, err error){
	// TODO: check member
	loger.Tracef("hoom.Server: Member(%s) trying connect", p.Mem.Id())
	cs := p.server.newServerConn(p.conn, p.Mem)
	if cs == nil {
		return NewSerror(fmt.Errorf("Member already exists")), nil
	}
	p.alive()
	loger.Debugf("hoom.Server: Member(%s) connected", p.Mem.Id())
	memid := p.Mem.Id()
	go func(){
		defer cs.free()
		for {
			select {
			case <-cs.conn.Context().Done():
				return
			case <-time.After(30 * time.Second):
				ctx, cancel := context.WithTimeout(cs.conn.Context(), 15 * time.Second)
				ping, err := cs.conn.PingWith(ctx)
				cancel()
				if err != nil {
					loger.Debugf("Ping member '%s' error: %v", memid, err)
					if errors.Is(err, context.Canceled) {
						return
					}
					return
				}
				_ = ping // TODO: Save client pings
			}
		}
	}()
	return
}

func (p *CjoinPkt)WriteTo(w encoding.Writer)(err error){
	if err = w.WriteUint32(p.RoomId); err != nil {
		return
	}
	return
}

func (p *CjoinPkt)ParseFrom(r encoding.Reader)(err error){
	if p.RoomId, err = r.ReadUint32(); err != nil {
		return
	}
	return
}

func (p *CjoinPkt)Ask()(res pio.PacketBase, err error){
	room, token, e := p.s.joinRoom(p.RoomId)
	if e != nil {
		res = NewSerror(e)
		return
	}
	res = &SjoinPkt{
		Room: room,
		Token: token,
		authServer: p.s.server.authServer,
	}
	return
}

func (p *CleavePkt)WriteTo(w encoding.Writer)(err error){
	if err = w.WriteUint32(p.RoomId); err != nil {
		return
	}
	return
}

func (p *CleavePkt)ParseFrom(r encoding.Reader)(err error){
	if p.RoomId, err = r.ReadUint32(); err != nil {
		return
	}
	return
}

func (p *CleavePkt)Ask()(res pio.PacketBase, err error){
	if e := p.s.leaveRoom(p.RoomId); e != nil {
		res = NewSerror(e)
		return
	}
	res = &SleavePkt{
		RoomId: p.RoomId,
	}
	return
}

func (p *CdialPkt)WriteTo(w encoding.Writer)(err error){
	if err = w.WriteString(p.MemId); err != nil {
		return
	}
	if err = p.Token.WriteTo(w); err != nil {
		return
	}
	return
}

func (p *CdialPkt)ParseFrom(r encoding.Reader)(err error){
	if p.MemId, err = r.ReadString(); err != nil {
		return
	}
	p.Token = new(RoomToken)
	if err = p.Token.ParseFrom(r); err != nil {
		return
	}
	return
}

func (p *CdialPkt)Ask()(res pio.PacketBase, err error){
	sc, ok := p.server.conns[p.MemId]
	if !ok {
		return NewSerror(fmt.Errorf("Member(%s) have not join this server", p.MemId)), nil
	}
	room, e := sc.checkToken(p.Token)
	if e != nil {
		return NewSerror(e), nil
	}
	roomid := room.Id()
	conn, e := sc.dial(roomid)
	if e != nil {
		return NewSerror(e), nil
	}
	p.alive()
	con := p.conn
	go func(){
		loger.Debug("hoom.Server: Waiting pio.Conn streamed")
		select {
		case <-con.StreamedDone():
		}
		loger.Trace("hoom.Server: pio.Conn streaming")
		rw, e := con.AsStream()
		if e != nil {
			conn.Close()
			con.Close()
			return
		}
		const bufSize = 1024 * 32 // 32 KB
		// TODO: use buf pool
		// TODO: count connections
		sc.putConn(roomid, con)
		loger.Debug("hoom.Server: Proxying pio.Conn and target")
		go func(){
			defer sc.popConn(roomid, con)
			err := <-ioProxy(conn, rw)
			loger.Trace("hoom.Server: ioProxy done")
			if err != nil && err != io.EOF {
				loger.Debugf("hoom.Server: ioProxy error: %v", err)
			}
		}()
	}()
	return
}


var (
	_ pio.PacketBase = (*SjoinPkt)(nil)
	_ pio.Packet     = (*SjoinBPkt)(nil)
	_ pio.PacketBase = (*SleavePkt)(nil)
	_ pio.Packet     = (*SleaveBPkt)(nil)
	_ pio.PacketBase = (*SerrorPkt)(nil)
)

func NewSerror(err error)(*SerrorPkt){
	return &SerrorPkt{
		Error: err.Error(),
	}
}

func (p *SjoinPkt)  PktId()(uint32){ return 0x91 }
func (p *SjoinBPkt) PktId()(uint32){ return 0x92 }
func (p *SleavePkt) PktId()(uint32){ return 0x93 }
func (p *SleaveBPkt)PktId()(uint32){ return 0x94 }
func (p *SerrorPkt) PktId()(uint32){ return 0x95 }

func (p *SjoinPkt)WriteTo(w encoding.Writer)(err error){
	if err = WriteRoom(w, p.Room, p.authServer); err != nil {
		return
	}
	if err = p.Token.WriteTo(w); err != nil {
		return
	}
	return
}

func (p *SjoinPkt)ParseFrom(r encoding.Reader)(err error){
	if p.Room, err = ParseRoom(r, p.authServer); err != nil {
		return
	}
	p.Token = new(RoomToken)
	if err = p.Token.ParseFrom(r); err != nil {
		return
	}
	return
}

func (p *SjoinBPkt)WriteTo(w encoding.Writer)(err error){
	if err = w.WriteUint32(p.RoomId); err != nil {
		return
	}
	if err = p.authServer.WriteMember(w, p.Mem); err != nil {
		return
	}
	return
}

func (p *SjoinBPkt)ParseFrom(r encoding.Reader)(err error){
	if p.RoomId, err = r.ReadUint32(); err != nil {
		return
	}
	if p.Mem, err = p.authServer.ParseMember(r); err != nil {
		return
	}
	return
}

func (p *SjoinBPkt)Trigger()(err error){
	r, ok := p.c.rooms[p.RoomId]
	if !ok {
		return fmt.Errorf("hoom.SjoinBPkt: Room %d does not connect", p.RoomId)
	}
	r.Put(p.Mem)
	loger.Tracef("hoom.Client(%p): Player '%s' joined room %s", p.c, p.Mem.Name(), r.Name())
	return
}

func (p *SleavePkt)WriteTo(w encoding.Writer)(err error){
	if err = w.WriteUint32(p.RoomId); err != nil {
		return
	}
	return
}

func (p *SleavePkt)ParseFrom(r encoding.Reader)(err error){
	if p.RoomId, err = r.ReadUint32(); err != nil {
		return
	}
	return
}

func (p *SleaveBPkt)WriteTo(w encoding.Writer)(err error){
	if err = w.WriteUint32(p.RoomId); err != nil {
		return
	}
	if err = w.WriteString(p.MemId); err != nil {
		return
	}
	return
}

func (p *SleaveBPkt)ParseFrom(r encoding.Reader)(err error){
	if p.RoomId, err = r.ReadUint32(); err != nil {
		return
	}
	if p.MemId, err = r.ReadString(); err != nil {
		return
	}
	return
}

func (p *SleaveBPkt)Trigger()(err error){
	r, ok := p.c.rooms[p.RoomId]
	if !ok {
		return fmt.Errorf("hoom.SjoinBPkt: Room %d does not connect", p.RoomId)
	}
	mem, ok := r.Pop(p.MemId)
	if !ok {
		return
	}
	loger.Tracef("hoom.Client(%p): Player '%s' leaved room %s", p.c, mem.Name(), r.Name())
	return
}

func (p *SerrorPkt)WriteTo(w encoding.Writer)(err error){
	if err = w.WriteString(p.Error); err != nil {
		return
	}
	return
}

func (p *SerrorPkt)ParseFrom(r encoding.Reader)(err error){
	if p.Error, err = r.ReadString(); err != nil {
		return
	}
	return
}
