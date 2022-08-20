
package hoom

import (
	"fmt"

	"github.com/kmcsr/go-pio/encoding"
	"github.com/kmcsr/go-pio"
)


type (
	CbindPkt struct{
		Mem *Member

		s *serverConn
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
		RoomId uint32

		s *serverConn
	}
	CclosePkt struct{
		RoomId uint32
		Reason string

		s *serverConn
	}
	CsendPkt struct{
		RoomId uint32
		Data []byte

		s *serverConn
	}
)

type (
	SjoinPkt struct{
		Room *Room
	}
	SjoinBPkt struct{
		RoomId uint32
		Mem *Member
	}
	SleavePkt struct{
		RoomId uint32
		Reason string
	}
	SleaveBPkt struct{
		RoomId uint32
		MemId uint32
	}
	SerrorPkt struct{
		Error string
	}
	SclosePkt struct{
		RoomId uint32
	}
	SsendPkt struct{
		RoomId uint32
		Data []byte
	}
)


var (
	_ pio.Packet    = (*CbindPkt)(nil)
	_ pio.PacketAsk = (*CjoinPkt)(nil)
	_ pio.PacketAsk = (*CleavePkt)(nil)
	_ pio.PacketAsk = (*CdialPkt)(nil)
	_ pio.PacketAsk = (*CclosePkt)(nil)
	_ pio.Packet    = (*CsendPkt)(nil)
)

func (p *CbindPkt) PktId()(uint32){ return 0x81 }
func (p *CjoinPkt) PktId()(uint32){ return 0x82 }
func (p *CleavePkt)PktId()(uint32){ return 0x83 }
func (p *CdialPkt) PktId()(uint32){ return 0x84 }
func (p *CclosePkt)PktId()(uint32){ return 0x85 }
func (p *CsendPkt) PktId()(uint32){ return 0x86 }

func (p *CbindPkt)WriteTo(w encoding.Writer)(err error){
	if err = p.Mem.WriteTo(w); err != nil {
		return
	}
	return
}

func (p *CbindPkt)ParseFrom(r encoding.Reader)(err error){
	p.Mem = new(Member)
	if err = p.Mem.ParseFrom(r); err != nil {
		return
	}
	return
}

func (p *CbindPkt)Trigger()(err error){
	if p.s.mem != nil {
		panic("Connection already binded")
	}
	p.s.mem = p.Mem
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
	room, e := p.s.joinRoom(p.RoomId)
	if e != nil {
		res = &SerrorPkt{
			Error: e.Error(),
		}
		return
	}
	res = &SjoinPkt{
		Room: room,
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
		res = &SerrorPkt{
			Error: e.Error(),
		}
		return
	}
	res = &SleavePkt{
		RoomId: p.RoomId,
	}
	return
}

func (p *CdialPkt)WriteTo(w encoding.Writer)(err error){
	if err = w.WriteUint32(p.RoomId); err != nil {
		return
	}
	return
}

func (p *CdialPkt)ParseFrom(r encoding.Reader)(err error){
	if p.RoomId, err = r.ReadUint32(); err != nil {
		return
	}
	return
}

func (p *CdialPkt)Ask()(res pio.PacketBase, err error){
	if e := p.s.dial(p.RoomId); e != nil {
		res = &SerrorPkt{
			Error: e.Error(),
		}
		return
	}
	return
}

func (p *CclosePkt)WriteTo(w encoding.Writer)(err error){
	if err = w.WriteUint32(p.RoomId); err != nil {
		return
	}
	return
}

func (p *CclosePkt)ParseFrom(r encoding.Reader)(err error){
	if p.RoomId, err = r.ReadUint32(); err != nil {
		return
	}
	return
}

func (p *CclosePkt)Ask()(res pio.PacketBase, err error){

	return
}

func (p *CsendPkt)WriteTo(w encoding.Writer)(err error){
	if err = w.WriteUint32(p.RoomId); err != nil {
		return
	}
	if err = w.WriteBytes(p.Data); err != nil {
		return
	}
	return
}

func (p *CsendPkt)ParseFrom(r encoding.Reader)(err error){
	if p.RoomId, err = r.ReadUint32(); err != nil {
		return
	}
	if p.Data, err = r.ReadBytes(); err != nil {
		return
	}
	return
}

func (p *CsendPkt)Trigger()(err error){
	conn := p.s.getRoomConn(p.RoomId)
	if conn == nil {
		panic(fmt.Errorf("Room(%d) not conntected", p.RoomId))
	}
	_, err = conn.Write(p.Data)
	return
}


var (
	_ pio.PacketBase = (*SjoinPkt)(nil)
	_ pio.Packet     = (*SjoinBPkt)(nil)
	_ pio.PacketBase = (*SleavePkt)(nil)
	_ pio.Packet     = (*SleaveBPkt)(nil)
	_ pio.PacketBase = (*SerrorPkt)(nil)
)

func (p *SjoinPkt)  PktId()(uint32){ return 0x91 }
func (p *SjoinBPkt) PktId()(uint32){ return 0x92 }
func (p *SleavePkt) PktId()(uint32){ return 0x93 }
func (p *SleaveBPkt)PktId()(uint32){ return 0x94 }
func (p *SerrorPkt) PktId()(uint32){ return 0x95 }
func (p *SclosePkt) PktId()(uint32){ return 0x96 }
func (p *SsendPkt)  PktId()(uint32){ return 0x97 }

func (p *SjoinPkt)WriteTo(w encoding.Writer)(err error){
	if err = p.Room.WriteTo(w); err != nil {
		return
	}
	return
}

func (p *SjoinPkt)ParseFrom(r encoding.Reader)(err error){
	p.Room = new(Room)
	if err = p.Room.ParseFrom(r); err != nil {
		return
	}
	return
}

func (p *SjoinBPkt)WriteTo(w encoding.Writer)(err error){
	if err = w.WriteUint32(p.RoomId); err != nil {
		return
	}
	if err = p.Mem.WriteTo(w); err != nil {
		return
	}
	return
}

func (p *SjoinBPkt)ParseFrom(r encoding.Reader)(err error){
	if p.RoomId, err = r.ReadUint32(); err != nil {
		return
	}
	p.Mem = new(Member)
	if err = p.Mem.ParseFrom(r); err != nil {
		return
	}
	return
}

func (p *SjoinBPkt)Trigger()(err error){
	panic("TODO")
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
	if err = w.WriteUint32(p.MemId); err != nil {
		return
	}
	return
}

func (p *SleaveBPkt)ParseFrom(r encoding.Reader)(err error){
	if p.RoomId, err = r.ReadUint32(); err != nil {
		return
	}
	if p.MemId, err = r.ReadUint32(); err != nil {
		return
	}
	return
}

func (p *SleaveBPkt)Trigger()(err error){
	panic("TODO")
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
