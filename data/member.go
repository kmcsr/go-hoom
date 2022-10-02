
package hoom_data

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

func (m *Member)SetName(name string){
	m.name = name
}

func (m *Member)Name()(string){
	return m.name
}
