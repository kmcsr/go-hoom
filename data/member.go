
package hoom_data

type Member struct{
	id string
	name string
}

func NewMember(id string, name string)(*Member){
	return &Member{
		id: id,
		name: name,
	}
}

func (m *Member)Id()(string){
	return m.id
}

func (m *Member)SetName(name string){
	m.name = name
}

func (m *Member)Name()(string){
	return m.name
}
