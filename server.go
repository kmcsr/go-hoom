
package hoom

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/kmcsr/go-pio"
)

type Server struct{
	Addr *net.TCPAddr

	owner *Member
	roomc uint32
	rooms map[uint32]*Room
	conns map[uint32]*serverConn
}

func (owner *Member)NewServer(addr *net.TCPAddr)(*Server){
	return &Server{
		Addr: addr,
		owner: owner,
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

func (s *Server)putConn(sc *serverConn)(bool){
	if _, ok := s.conns[sc.mem.Id()]; ok {
		return false
	}
	s.conns[sc.mem.Id()] = sc
	return true
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
	conns map[uint32]net.Conn
}

func (s *serverConn)checkBinded(){
	if s.mem == nil {
		panic("Connection need to bind a member")
	}
}

func (s *serverConn)free(){
	for _, r := range s.server.rooms {
		r.pop(s.mem.Id())
	}
	s.mem = nil
	s.conn.Close()
	for _, c := range s.conns {
		c.Close()
	}
}

func (s *serverConn)joinRoom(id uint32)(r *Room, err error){
	s.checkBinded()
	r, ok := s.server.rooms[id]
	if !ok {
		return nil, fmt.Errorf("Room(%d) not exists", id)
	}
	if !r.put(s.mem) {
		return nil, fmt.Errorf("Member(%d) already exists", s.mem.Id())
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
	s.checkBinded()
	r, ok := s.server.rooms[id]
	if !ok {
		return fmt.Errorf("Room(%d) is not connected", id)
	}
	r.pop(s.mem.Id())
	return s.closeConn(id)
}

func (s *serverConn)getConn(id uint32)(c net.Conn){
	s.checkBinded()
	return s.conns[id]
}

func (s *serverConn)dial(id uint32)(ses uint32, err error){
	s.checkBinded()
	r, ok := s.server.rooms[id]
	if !ok {
		err = fmt.Errorf("Room(%d) not exists", id)
		return
	}
	var conn *net.TCPConn
	if conn, err = net.DialTCP("tcp", nil, r.Target()); err != nil {
		return
	}
	ses = 0
	for {
		ses++
		if _, ok := s.conns[ses]; !ok {
			break
		}
	}
	go func(){
		defer conn.Close()
		var (
			buf = make([]byte, 1024 * 128) // 128 KB
			n int
			err error
		)
		for {
			if n, err = conn.Read(buf); err != nil {
				if err == io.EOF || errors.Is(err, net.ErrClosed) {
					err = nil
				}
				break
			}
			if err = s.conn.Send(&SsendPkt{
				ConnId: ses,
				Data: buf[:n],
			}); err != nil {
				break
			}
		}
		if err != nil {
			// TODO: Logger
			println("error", err.Error())
		}
	}()
	s.conns[ses] = conn
	return
}

func (s *serverConn)closeConn(id uint32)(err error){
	s.checkBinded()
	c, ok := s.conns[id]
	if !ok {
		return
	}
	delete(s.conns, id)
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
	defer s.free()
	return s.conn.Serve()
}

func (s *Server)ListenAndServe()(err error){
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
		// TODO: Logger
		fmt.Println("debug", "Client accepted:", c.RemoteAddr())
		cs := &serverConn{
			server: s,
			conn: pio.NewConn(c, c),
			conns: make(map[uint32]net.Conn),
		}
		go cs.serve()
		go func(){
			defer cs.conn.Close()
			for {
				select {
				case <-cs.conn.Context().Done():
					return
				case <-time.After(10 * time.Second):
					ctx, cancel := context.WithTimeout(cs.conn.Context(), 15 * time.Second)
					ping, err := cs.conn.PingWith(ctx)
					cancel()
					if err != nil {
						if errors.Is(err, context.Canceled) {
							return
						}
						// TODO: Logger
						fmt.Println("debug", "Ping error:", err)
						return
					}
					_ = ping // TODO: Save client pings
				}
			}
		}()
	}
	return nil
}

