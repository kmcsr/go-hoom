
package hoom

import (
	"io"
	"net"

	"github.com/google/uuid"
	"github.com/kmcsr/go-pio/encoding"
	hdata "github.com/kmcsr/go-hoom/data"
)

type Desc = hdata.Desc

type Room = hdata.Room

var NewRoom = hdata.NewRoom

func WriteRoom(w encoding.Writer, r *Room)(err error){
	if err = w.WriteUint32(r.Id()); err != nil {
		return
	}
	if err = WriteMember(w, r.Owner()); err != nil {
		return
	}
	tid := r.TypeId()
	if _, err = w.Write(tid[:]); err != nil {
		return
	}
	if err = w.WriteString(r.Name()); err != nil {
		return
	}
	mems := r.Members()
	if err = w.WriteUint32((uint32)(len(mems))); err != nil {
		return
	}
	for _, m := range mems {
		if err = WriteMember(w, m); err != nil {
			return
		}
	}
	return
}

func ParseRoom(r encoding.Reader)(rm *Room, err error){
	var (
		id uint32
		name string
		mem *Member
		tid uuid.UUID
	)
	if id, err = r.ReadUint32(); err != nil {
		return
	}
	if mem, err = ParseMember(r); err != nil {
		return
	}
	if _, err = io.ReadFull(r, tid[:]); err != nil {
		return
	}
	if name, err = r.ReadString(); err != nil {
		return
	}
	rm = NewRoom(id, mem, tid, name)
	var n uint32
	if n, err = r.ReadUint32(); err != nil {
		return
	}
	for i := (uint32)(0); i < n; i++ {
		if mem, err = ParseMember(r); err != nil {
			return
		}
		rm.Put(mem)
	}
	return
}

type ServerRoom struct{
	*Room
	server *Server
	target *net.TCPAddr
}

func (r *ServerRoom)Server()(*Server){
	return r.server
}

func (r *ServerRoom)Target()(*net.TCPAddr){
	return r.target
}

func (r *ServerRoom)Kick(id uint32, reason string)(err error){
	return r.server.Kick(r.Id(), id, reason)
}

type RoomToken struct{
	RoomId uint32
	MemId uint32
	Token uint64
	Sign []byte
}

func (t *RoomToken)WriteTo(w encoding.Writer)(err error){
	if err = w.WriteUint32(t.RoomId); err != nil {
		return
	}
	if err = w.WriteUint32(t.MemId); err != nil {
		return
	}
	if err = w.WriteUint64(t.Token); err != nil {
		return
	}
	if err = w.WriteBytes(t.Sign); err != nil {
		return
	}
	return
}

func (t *RoomToken)ParseFrom(r encoding.Reader)(err error){
	if t.RoomId, err = r.ReadUint32(); err != nil {
		return
	}
	if t.MemId, err = r.ReadUint32(); err != nil {
		return
	}
	if t.Token, err = r.ReadUint64(); err != nil {
		return
	}
	if t.Sign, err = r.ReadBytes(); err != nil {
		return
	}
	return
}

