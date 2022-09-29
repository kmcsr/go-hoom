
package hoom

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/kmcsr/go-pio/encoding"
	"github.com/kmcsr/go-pio"
)

var TokenNotValid = errors.New("Token not valid")

type Server struct{
	Addr *net.TCPAddr
	Listener *net.TCPListener

	owner *Member
	roomc uint32
	rooms map[uint32]*Room
	conns map[uint32]*serverConn
}

func (m *AuthedMember)NewServer(addr *net.TCPAddr)(*Server){
	return &Server{
		Addr: addr,
		owner: m.GetMem(),
		rooms: make(map[uint32]*Room),
		conns: make(map[uint32]*serverConn),
	}
}

func (s *Server)NewRoom(name string, target *net.TCPAddr)(r *Room){
	s.roomc++
	id := s.roomc
	for {
		if _, ok := s.rooms[id]; !ok {
			break
		}
		id++
	}
	r = &Room{
		id: id,
		name: name,
		owned: true,
		server: s,
		target: target,
		owner: s.owner,
		members: make(map[uint32]*Member),
	}
	s.rooms[id] = r
	return
}

func (s *Server)PopRoom(id uint32)(r *Room){
	r, ok := s.rooms[id]
	if ok {
		delete(s.rooms, id)
	}
	return
}

func (s *Server)Rooms()(rooms []*Room){
	rooms = make([]*Room, 0, len(s.rooms))
	for _, r := range s.rooms {
		rooms = append(rooms, r)
	}
	return
}

func (s *Server)GetRoom(id uint32)(r *Room){
	return s.rooms[id]
}

func (s *Server)PopConn(mid uint32)(ok bool){
	var sc *serverConn
	if sc, ok = s.conns[mid]; ok {
		delete(s.conns, mid)
		sc.free()
	}
	return
}

func (s *Server)Kick(rid uint32, mid uint32, reason string)(err error){
	sc, ok := s.conns[mid]
	if !ok {
		return fmt.Errorf("Member(%d) connection is not exists", mid)
	}
	return sc.KickRoom(rid, reason)
}

type serverConn struct{
	server *Server
	conn *pio.Conn
	mem *Member
	conns map[uint32][]*pio.Conn
	tokens map[uint32]uint64
}

func (s *Server)newServerConn(conn *pio.Conn, mem *Member)(sc *serverConn){
	if _, ok := s.conns[mem.Id()]; ok {
		return nil
	}
	sc = &serverConn{
		server: s,
		conn: conn,
		mem: mem,
		conns: make(map[uint32][]*pio.Conn),
		tokens: make(map[uint32]uint64),
	}
	sc.initPackets()
	s.conns[mem.Id()] = sc
	return
}

func (s *serverConn)free(){
	if s.mem == nil {
		return
	}
	delete(s.server.conns, s.mem.Id())
	for _, r := range s.server.rooms {
		r.pop(s.mem.Id())
	}
	s.mem = nil
	s.conn.Close()
	s.conn = nil
	for _, cc := range s.conns {
		for _, c := range cc {
			c.Close()
		}
	}
	s.conns = nil
}

func (s *serverConn)joinRoom(id uint32)(r *Room, token *RoomToken, err error){
	r, ok := s.server.rooms[id]
	if !ok {
		return nil, nil, fmt.Errorf("Room(%d) not exists", id)
	}
	if !r.put(s.mem) {
		return nil, nil, fmt.Errorf("Member(%d) already exists", s.mem.Id())
	}
	token = &RoomToken{
		RoomId: id,
		MemId: s.mem.Id(),
		Token: RandUint64(),
	}
	// TODO: sign this token
	s.tokens[token.RoomId] = token.Token
	return
}

func (s *serverConn)checkToken(token *RoomToken)(r *Room, err error){
	if token.MemId != s.mem.Id() {
		return nil, TokenNotValid
	}
	tk, ok := s.tokens[token.RoomId]
	if !ok {
		return nil, TokenNotValid
	}
	if token.Token != tk {
		return nil, TokenNotValid
	}
	// TODO: check the signature
	r, ok = s.server.rooms[token.RoomId]
	if !ok {
		return nil, TokenNotValid
	}
	return
}

func (s *serverConn)KickRoom(id uint32, reason string)(err error){
	if err = s.leaveRoom(id); err != nil {
		return
	}
	if err = s.conn.Send(&SleavePkt{RoomId: id, Reason: reason}); err != nil {
		return
	}
	return
}

func (s *serverConn)leaveRoom(id uint32)(err error){
	r, ok := s.server.rooms[id]
	if !ok {
		return fmt.Errorf("Room(%d) is not exists", id)
	}
	if _, ok := s.tokens[id]; !ok {
		return fmt.Errorf("Room(%d) doesn't joined", id)
	}
	delete(s.tokens, id)
	r.pop(s.mem.Id())
	if cc, ok := s.conns[id]; ok {
		delete(s.conns, id)
		for _, c := range cc {
			c.Close()
		}
	}
	return
}

func (s *serverConn)dial(id uint32)(conn net.Conn, err error){
	r, ok := s.server.rooms[id]
	if !ok {
		err = fmt.Errorf("Room(%d) not exists", id)
		return
	}
	var tcon *net.TCPConn
	if tcon, err = net.DialTCP("tcp", nil, r.Target()); err != nil {
		return
	}
	conn = tcon
	return
}

func (s *serverConn)putConn(id uint32, conn *pio.Conn){
	s.conns[id] = append(s.conns[id], conn)
}

func (s *serverConn)popConn(id uint32, conn *pio.Conn){
	conns := s.conns[id]
	for i, c := range conns {
		if c == conn {
			conns[i] = conns[len(conns) - 1]
			conns = conns[:len(conns) - 1]
			break
		}
	}
	s.conns[id] = conns
}

func (s *serverConn)initPackets(){
	s.conn.AddPacket(func()(pio.PacketBase){ return &CjoinPkt {s: s} })
	s.conn.AddPacket(func()(pio.PacketBase){ return &CleavePkt{s: s} })
}

func (s *Server)Listen()(err error){
	if s.Listener, err = net.ListenTCP("tcp", s.Addr); err != nil {
		return
	}
	return
}

func (s *Server)Shutdown()(err error){
	if s.Listener == nil {
		return
	}
	if err = s.Listener.Close(); err != nil {
		return
	}
	s.Listener = nil
	return
}

func (s *Server)ListenAddr()(*net.TCPAddr){
	if s.Listener == nil {
		return s.Addr
	}
	return s.Listener.Addr().(*net.TCPAddr)
}

func (s *Server)serveConn(c net.Conn){
	alivectx, cancel := context.WithTimeout(context.Background(), 10 * time.Second)
	conn := pio.NewConn(c, c)
	conn.AddPacket(func()(pio.PacketBase){ return &CbindPkt{
		server: s,
		conn: conn,
		alive: cancel,
	} })
	conn.AddPacket(func()(pio.PacketBase){ return &CdialPkt{
		server: s,
		conn: conn,
		alive: cancel,
	} })
	conn.OnPktNotFound = func(id uint32, body encoding.Reader){
		fmt.Println("debug", "Unexpected packet id:", id) // TODO: Logger.WARN
	}
	go conn.Serve()
	go func(){
		defer cancel()
		select {
		case <-alivectx.Done():
			if errors.Is(alivectx.Err(), context.Canceled) {
				return
			}
		case <-conn.Context().Done():
		}
		conn.Close()
	}()
}

func (s *Server)Serve()(err error){
	listener := s.Listener
	if listener == nil {
		panic("Server need a listener")
	}
	for {
		var c net.Conn
		c, err = listener.Accept()
		if err != nil {
			return
		}
		// TODO: Logger
		fmt.Println("debug", "Client accepted:", c.RemoteAddr())

		s.serveConn(c)
	}
	return nil
}

func (s *Server)ListenAndServe()(err error){
	if err = s.Listen(); err != nil {
		return
	}
	return s.Serve()
}
