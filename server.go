
package hoom

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/google/uuid"
	"github.com/kmcsr/go-pio/encoding"
	"github.com/kmcsr/go-pio"
)

var TokenNotValid = errors.New("Token not valid")

type Server struct{
	Addr *net.TCPAddr
	Listener *net.TCPListener

	Handshakers []Handshaker

	owner *Member
	roomc uint32
	rooms map[uint32]*ServerRoom
	conns map[uint32]*serverConn
}

func (m *AuthedMember)NewServer(addr *net.TCPAddr)(*Server){
	return &Server{
		Addr: addr,
		owner: m.GetMem(),
		rooms: make(map[uint32]*ServerRoom),
		conns: make(map[uint32]*serverConn),
	}
}

func (s *Server)AddHandshaker(hs Handshaker)(*Server){
	if hs.Id() == NoneConnId {
		panic("handshaker's id cannot be zero")
	}
	s.Handshakers = append(s.Handshakers, hs)
	return s
}

func (s *Server)PopHandshaker(id byte){
	for i, hs := range s.Handshakers {
		if hs.Id() == id {
			copy(s.Handshakers[i:], s.Handshakers[i + 1:])
			s.Handshakers = s.Handshakers[:len(s.Handshakers) - 1]
			break
		}
	}
}

func (s *Server)handshake(c io.ReadWriteCloser)(rw io.ReadWriteCloser, err error){
	r := encoding.WrapReader(c)
	w := encoding.WrapWriter(c)
	for _, hs := range s.Handshakers {
		if err = w.WriteByte(hs.Id()); err != nil {
			return
		}
		var flag bool
		if flag, err = r.ReadBool(); err != nil {
			return
		}
		if flag {
			loger.Tracef("hoom.Server: Using handshaker (0x%02x)", hs.Id())
			return hs.HandServer(c)
		}
	}
	if err = w.WriteByte(NoneConnId); err != nil {
		return
	}
	return nil, NoCommonHandshaker
}

func (s *Server)NewRoom(name string, target *net.TCPAddr)(r *ServerRoom){
	s.roomc++
	id := s.roomc
	for {
		if _, ok := s.rooms[id]; !ok {
			break
		}
		id++
	}
	loger.Tracef("hoom.Server: Creating room id=%d name=%s target=%v", id, name, target)
	r = &ServerRoom{
		Room: NewRoom(id, s.owner, uuid.Nil, name),
		server: s,
		target: target,
	}
	s.rooms[id] = r
	return
}

func (s *Server)PopRoom(id uint32)(r *ServerRoom){
	r, ok := s.rooms[id]
	if ok {
		delete(s.rooms, id)
	}
	return
}

func (s *Server)Rooms()(rooms []*ServerRoom){
	rooms = make([]*ServerRoom, 0, len(s.rooms))
	for _, r := range s.rooms {
		rooms = append(rooms, r)
	}
	return
}

func (s *Server)GetRoom(id uint32)(r *ServerRoom){
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

func (s *Server)signToken(token *RoomToken)(err error){
	return
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
	loger.Trace("hoom.serverConn: Cleaning")
	delete(s.server.conns, s.mem.Id())
	for _, r := range s.server.rooms {
		r.Pop(s.mem.Id())
	}
	s.mem = nil
	s.conn.Close()
	for _, cc := range s.conns {
		for _, c := range cc {
			c.Close()
		}
	}
	s.conns = nil
}

func (s *serverConn)joinRoom(id uint32)(r *Room, token *RoomToken, err error){
	sr, ok := s.server.rooms[id]
	if !ok {
		return nil, nil, fmt.Errorf("Room(%d) not exists", id)
	}
	r = sr.Room
	if !sr.Put(s.mem) {
		return nil, nil, fmt.Errorf("Member(%d) already exists", s.mem.Id())
	}
	token = &RoomToken{
		RoomId: id,
		MemId: s.mem.Id(),
		Token: randUint64(),
	}
	if err = s.server.signToken(token); err != nil {
		return
	}
	s.tokens[token.RoomId] = token.Token
	return
}

func (s *serverConn)checkToken(token *RoomToken)(r *ServerRoom, err error){
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
	loger.Debugf("hoom.serverConn: Client is leaving room %d", id)
	delete(s.tokens, id)
	r.Pop(s.mem.Id())
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
	loger.Tracef("hoom.serverConn: a connection joined %d", id)
	s.conns[id] = append(s.conns[id], conn)
}

func (s *serverConn)popConn(id uint32, conn *pio.Conn){
	loger.Tracef("hoom.serverConn: a connection leaved %d", id)
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
	loger.Debugf("hoom.Server is listing %v", s.Listener.Addr())
	return
}

func (s *Server)Shutdown()(err error){
	if s.Listener == nil {
		return
	}
	loger.Debug("hoom.Server is shuting down")
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

func (s *Server)serveConn(c io.ReadWriteCloser){
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
		loger.Warn("hoom.Server: Unexpected packet id:", id)
	}
	go conn.Serve()
	go func(){
		defer cancel()
		select {
		case <-alivectx.Done():
			if errors.Is(alivectx.Err(), context.Canceled) {
				return
			}
			loger.Debug("hoom.serverConn: Connection init activity timeout")
		case <-conn.Context().Done():
			loger.Debug("hoom.serverConn: Connection init activity canceled")
		}
		conn.Close()
	}()
}

func (s *Server)Serve()(err error){
	listener := s.Listener
	if listener == nil {
		panic("Server need a listener")
	}
	loger.Debug("Serving hoom.Server")
	for {
		var c net.Conn
		c, err = listener.Accept()
		if err != nil {
			return
		}
		loger.Tracef("Client '%v' handshaking", c.RemoteAddr())
		var rw io.ReadWriteCloser
		rw, err = s.handshake(c)
		if err != nil {
			loger.Debugf("Client '%v' handshake error: %v", c.RemoteAddr(), err)
			c.Close()
			continue
		}
		loger.Debugf("Client accepted: %v", c.RemoteAddr())

		s.serveConn(rw)
	}
	return nil
}

func (s *Server)ListenAndServe()(err error){
	if err = s.Listen(); err != nil {
		return
	}
	return s.Serve()
}
