
package hoom

import (
	"fmt"
	"net"

	"github.com/kmcsr/go-pio"
)

type serverRoom struct{
	*Room
	conns []*serverConn
}

func (r *serverRoom)put(s *serverConn)(ok bool){
	if ok = r.Room.put(s.mem); ok {
		r.conns = append(r.conns, s)
	}
	return 
}

func (r *serverRoom)pop(s *serverConn)(ok bool){
	if _, ok = r.Room.pop(s.mem.Id()); ok {
		for i, c := range r.conns {
			if c == s {
				r.conns[i] = r.conns[len(r.conns) - 1]
				r.conns = r.conns[:len(r.conns) - 1]
			}
		}
	}
	return
}

type Server struct{
	Addr *net.TCPAddr

	owner *Member
	rooms map[uint32]*serverRoom
}

func NewServer(addr *net.TCPAddr, owner *Member)(*Server){
	return &Server{
		Addr: addr,
		owner: owner,
		rooms: make(map[uint32]*serverRoom),
	}
}

func (s *Server)PutRoom(r *Room){
	if _, ok := s.rooms[r.Id()]; ok {
		panic(fmt.Errorf("Room(%d) already exists", r.Id()))
	}
	s.rooms[r.Id()] = &serverRoom{Room: r}
}

func (s *Server)GetRoom(id uint32)(r *Room){
	r0, ok := s.rooms[id]
	if ok {
		r = r0.Room
	}
	return
}

func (s *Server)PopRoom(id uint32)(r *Room){
	r0, ok := s.rooms[id]
	if !ok {
		return
	}
	delete(s.rooms, id)
	r = r0.Room
	for _, sc := range r0.conns {
		sc.leaveRoom(id)
	}
	return
}

type serverConn struct{
	server *Server
	conn *pio.Conn
	mem *Member
	rconn map[uint32]net.Conn
}

func (s *serverConn)checkBinded(){
	if s.mem == nil {
		panic("Connection need to bind a member")
	}
}

func (s *serverConn)free(){
	for _, r := range s.server.rooms {
		r.pop(s)
	}
	s.mem = nil
	s.conn.Close()
	for _, c := range s.rconn {
		c.Close()
	}
}

func (s *serverConn)joinRoom(id uint32)(r *Room, err error){
	s.checkBinded()
	r0, ok := s.server.rooms[id]
	if !ok {
		return nil, fmt.Errorf("Room(%d) not exists", id)
	}
	if !r0.put(s) {
		return nil, fmt.Errorf("Member(%d) already exists", s.mem.Id())
	}
	r = r0.Room
	return
}

func (s *serverConn)LeaveRoom(id uint32, reason string)(err error){
	if err = s.leaveRoom(id); err != nil {
		return
	}
	if err = s.conn.Send(&SleavePkt{RoomId: id, Reason: reason}); err != nil {
		return
	}
	return
}

func (s *serverConn)leaveRoom(id uint32)(err error){
	s.checkBinded()
	r, ok := s.server.rooms[id]
	if !ok {
		return fmt.Errorf("Room(%d) is not connected", id)
	}
	s.closeConn(id)
	r.pop(s)
	return
}

func (s *serverConn)getRoomConn(id uint32)(c net.Conn){
	s.checkBinded()
	return s.rconn[id]
}

func (s *serverConn)dial(id uint32)(err error){
	s.checkBinded()
	r, ok := s.server.rooms[id]
	if !ok {
		return fmt.Errorf("Room(%d) not exists", id)
	}
	if _, ok := s.rconn[id]; ok {
		return fmt.Errorf("Room(%d) is already connected", id)
	}
	var conn *net.TCPConn
	if conn, err = net.DialTCP("tcp", nil, r.target); err != nil {
		return
	}
	s.rconn[id] = conn
	return
}

func (s *serverConn)closeConn(id uint32)(err error){
	s.checkBinded()
	c, ok := s.rconn[id]
	if !ok {
		return
	}
	delete(s.rconn, id)
	return c.Close()
}

func (s *serverConn)initConn(){
	s.conn.AddPacket(func()(pio.PacketBase){ return &CbindPkt {s: s} })
	s.conn.AddPacket(func()(pio.PacketBase){ return &CjoinPkt {s: s} })
	s.conn.AddPacket(func()(pio.PacketBase){ return &CleavePkt{s: s} })
	s.conn.AddPacket(func()(pio.PacketBase){ return &CdialPkt {s: s} })
	s.conn.AddPacket(func()(pio.PacketBase){ return &CclosePkt{s: s} })
	s.conn.AddPacket(func()(pio.PacketBase){ return &CsendPkt {s: s} })
}

func (s *serverConn)serve()(err error){
	s.initConn()
	return s.conn.Serve()
}

func (s *Server)Serve()(err error){
	var listener *net.TCPListener
	listener, err = net.ListenTCP("tcp", s.Addr)
	if err != nil {
		return
	}
	if s.Addr == nil {
		s.Addr = listener.Addr().(*net.TCPAddr)
	}
	for {
		var c net.Conn
		c, err = listener.Accept()
		if err != nil {
			return
		}
		cs := &serverConn{
			server: s,
			conn: pio.NewConn(c, c),
			rconn: make(map[uint32]net.Conn),
		}
		go cs.serve()
	}
	return nil
}

