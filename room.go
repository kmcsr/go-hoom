
package hoom

import (
	"net"

	"github.com/kmcsr/go-pio/encoding"
)

type Member struct{
	id uint32
	name string
}

func NewMember(id uint32, name string)(*Member){
	return &Member{
		id: id,
		name: name,
	}
}

func (m *Member)Id()(uint32){
	return m.id
}

func (m *Member)Name()(string){
	return m.name
}

func (m *Member)WriteTo(w encoding.Writer)(err error){
	if err = w.WriteUint32(m.id); err != nil {
		return
	}
	if err = w.WriteString(m.name); err != nil {
		return
	}
	return
}

func (m *Member)ParseFrom(r encoding.Reader)(err error){
	if m.id, err = r.ReadUint32(); err != nil {
		return
	}
	if m.name, err = r.ReadString(); err != nil {
		return
	}
	return
}


type Room struct{
	id uint32
	name string

	owned bool
	server *Server
	target *net.TCPAddr
	owner *Member
	members map[uint32]*Member
}

func NewRoom(id uint32, name string, server *Server, target *net.TCPAddr)(*Room){
	return &Room{
		id: id,
		name: name,
		owned: true,
		server: server,
		target: target,
		owner: server.owner,
		members: make(map[uint32]*Member),
	}
}

func (r *Room)Id()(uint32){
	return r.id
}

func (r *Room)Name()(string){
	return r.name
}

func (r *Room)SetName(name string){
	r.name = name
}

func (r *Room)Owner()(*Member){
	return r.owner
}

func (r *Room)Members()(mems []*Member){
	mems = make([]*Member, 0, len(r.members))
	for _, m := range r.members {
		mems = append(mems, m)
	}
	return
}

func (r *Room)MemLen()(int){
	return len(r.members)
}

func (r *Room)checkOwned(){
	if !r.owned {
		panic("You not owned this room")
	}
}

func (r *Room)Server()(*Server){
	r.checkOwned()
	return r.server
}

func (r *Room)Target()(*net.TCPAddr){
	r.checkOwned()
	return r.target
}

func (r *Room)WriteTo(w encoding.Writer)(err error){
	r.checkOwned()
	if err = w.WriteUint32(r.id); err != nil {
		return
	}
	if err = w.WriteString(r.name); err != nil {
		return
	}
	if err = r.owner.WriteTo(w); err != nil {
		return
	}
	return
}

func (r *Room)ParseFrom(rd encoding.Reader)(err error){
	if r.owned {
		panic("Room is owned")
	}
	if r.owner != nil {
		panic("Room is inited")
	}
	if r.id, err = rd.ReadUint32(); err != nil {
		return
	}
	if r.name, err = rd.ReadString(); err != nil {
		return
	}
	owner := new(Member)
	if err = r.owner.ParseFrom(rd); err != nil {
		return
	}
	r.owner = owner
	r.members = make(map[uint32]*Member)
	return
}

func (r *Room)put(m *Member)(bool){
	r.checkOwned()
	if m.Id() == r.owner.Id() {
		panic("Cannot put owner member")
	}
	if  _, ok := r.members[m.Id()]; ok {
		return false
	}
	r.members[m.Id()] = m
	return true
}

func (r *Room)pop(id uint32)(m *Member, ok bool){
	if id == r.owner.Id() {
		panic("Cannot pop owner")
	}
	m, ok = r.members[id]
	if ok {
		delete(r.members, id)
	}
	return
}

func (r *Room)GetMember(id uint32)(m *Member){
	return r.members[id]
}

func (r *Room)Kick(id uint32, reason string)(err error){
	r.checkOwned()
	return r.server.Kick(r.id, id, reason)
}

