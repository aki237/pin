package pinlib

import (
	"fmt"
	"net"
	"sync"

	"github.com/songgao/water"
)

// Server struct contains all fields for exchanging packets to the client through a TCP connection
type Server struct {
	server net.Listener
	iface  *water.Interface
}

// NewServer method is used to create a new server struct with a given listening address
func NewServer(addr string, iface *water.Interface) (*Server, error) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	return &Server{server: ln, iface: iface}, nil
}

// Start method accepts TCP connections from a client and starts the packet exchange from the local tunneling interface to the remote client
// This also makes Server struct to satisfy the pinlib.Peer interface.
func (s *Server) Start() error {
	wg := &sync.WaitGroup{}

	for {
		conn, err := s.server.Accept()
		if err != nil {
			fmt.Println(err)
			continue
		}

		ex := &Exchanger{conn: conn, iface: s.iface}
		wg.Add(1)
		go ex.Start(wg)
	}

	wg.Wait()

	return nil
}
