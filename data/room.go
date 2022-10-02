
package hoom_data

import (
	"github.com/google/uuid"
)

type Desc struct{
	Name string
	Value string
}

type Room struct{
	id uint32
	owner *Member
	typeId uuid.UUID 
	name string
	desc []Desc
	members map[uint32]*Member
}

func NewRoom(id uint32, owner *Member, typeId uuid.UUID, name string)(*Room){
	return &Room{
		id: id,
		owner: owner,
		typeId: typeId,
		name: name,
		members: make(map[uint32]*Member),
	}
}

func (r *Room)Id()(uint32){
	return r.id
}

func (r *Room)Owner()(*Member){
	return r.owner
}

func (r *Room)TypeId()(uuid.UUID){
	return r.typeId
}

func (r *Room)Name()(string){
	return r.name
}

func (r *Room)SetName(name string){
	r.name = name
}

func (r *Room)Desc()([]Desc){
	return r.desc
}

func (r *Room)SetDesc(desc []Desc){
	r.desc = desc
}

func (r *Room)AddDesc(desc Desc){
	r.desc = append(r.desc, desc)
}

func (r *Room)Members()(mems []*Member){
	mems = make([]*Member, 0, len(r.members))
	for _, m := range r.members {
		mems = append(mems, m)
	}
	return
}

func (r *Room)MemCount()(int){
	return len(r.members)
}

func (r *Room)GetMember(id uint32)(m *Member){
	return r.members[id]
}

func (r *Room)Put(m *Member)(bool){
	if m.Id() == r.owner.Id() {
		panic("Cannot put room owner to the room")
	}
	if _, ok := r.members[m.Id()]; !ok {
		r.members[m.Id()] = m
		return true
	}
	return false
}

func (r *Room)Pop(id uint32)(m *Member, ok bool){
	if id == r.owner.Id() {
		panic("Cannot pop owner from the room")
	}
	if m, ok = r.members[id]; ok {
		delete(r.members, id)
	}
	return
}

