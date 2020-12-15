package pinlib

import (
	"fmt"
	"io"
	"net"
	"sync"
)

// connSession stores the pipe and the connection resources for a specific client.
type connSession struct {
	filteredWr io.WriteCloser
	filteredRd io.ReadCloser
	conn       net.Conn
}

// Server struct contains all fields for exchanging packets to the client through a TCP connection
type Server struct {
	gw        *net.IPNet
	dhcpMutex *sync.Mutex
	lastIP    net.IP
	server    net.Listener
	iface     io.ReadWriteCloser
	protocol  string
	motd      [128]byte

	secret      [32]byte
	closeSignal chan bool

	clientsMutex *sync.Mutex
	clients      map[string]connSession
}

// NewServer method is used to create a new server struct with a given listening address
func NewServer(ip net.IP, port int, protocol string, iface io.ReadWriteCloser, gw *net.IPNet, secret [32]byte) (*Server, error) {
	srv := &Server{
		iface: iface, gw: gw, secret: secret,
		lastIP:       gw.IP,
		dhcpMutex:    &sync.Mutex{},
		clientsMutex: &sync.Mutex{},
		clients:      make(map[string]connSession),
		protocol:     protocol,
		closeSignal:  make(chan bool),
	}
	var err error

	switch protocol {
	case "tcp":
		srv.server, err = net.ListenTCP("tcp", &net.TCPAddr{IP: ip, Port: port})
	case "udp":
		//srv.server, err = net.ListenTCP("udp", &net.UDPAddr{IP: ip, Port: port})
		err = fmt.Errorf("Not implemented")
	}
	if err != nil {
		return nil, err
	}

	return srv, nil
}

// SetMotd is used to set the "message of the day" which
// will be sent during the final handshake step.
func (s *Server) SetMotd(data [128]byte) {
	s.motd = data
}

func (s *Server) nextIP() (net.IP, bool) {
	s.dhcpMutex.Lock()
	for i := len(s.lastIP) - 1; i >= 0; i-- {
		s.lastIP[i]++
		_, ok := s.clients[string(s.lastIP)]
		if s.lastIP[i] != 0 && !ok {
			break
		}
	}
	s.dhcpMutex.Unlock()
	if !s.gw.Contains(s.lastIP) {
		return nil, false
	}
	return s.lastIP.To4(), true
}

// Start method accepts TCP connections from a client and starts the packet exchange from the local tunneling interface to the remote client
// This also makes Server struct to satisfy the pinlib.Peer interface.
func (s *Server) Start() error {
	switch s.protocol {
	case "tcp":
		return s.StartTCP()
	case "udp":
		return fmt.Errorf("not implemented")
	default:
		return fmt.Errorf("unsupported protocol")
	}
}

// StartTCP starts the TCP VPN server
func (s *Server) StartTCP() error {
	wg := &sync.WaitGroup{}
	muxQueueCount := 1
	wg.Add(muxQueueCount)
	for i := 0; i < muxQueueCount; i++ {
		go func(_wg *sync.WaitGroup) {
			s.MuxTCP()
			_wg.Done()
		}(wg)
	}
	for {
		conn, err := s.server.Accept()
		if err != nil {
			s.Close()
			wg.Wait()
			return err
		}
		go s.HandleTCPConnection(NewCryptoConn(conn, s.secret))
	}
}

// HandleTCPConnection runs the VPN session for the specific TCP client.
func (s *Server) HandleTCPConnection(c net.Conn) {
	ip, err := s.HandshakeTCP(c)
	if err != nil {
		c.Close()
		fmt.Printf("Client closing (%s) : %s", ip, err)
		return
	}

	session := s.AddClient(ip, c)

	ex := &Exchanger{
		conn: &CallbackConn{
			ReadWriteCloser: c,
			ip:              ip,
			Callback:        func() { s.RemoveClient(ip) },
		},
		iface: &InterfaceSession{
			Reader: session.filteredRd,
			Writer: s.iface,
		},
	}

	ex.Start()
}

// HandshakeTCP conducts the handshake for IP negotiation for TCP clients.
func (s *Server) HandshakeTCP(c net.Conn) (net.IP, error) {
	var msg *HandshakeMessage
	var err error

	msg, err = getMessage(c)
	if err != nil {
		return nil, err
	}

	if msg.Type != IPRequest {
		return nil, fmt.Errorf("invalid handshake sequence, expected: %d, got: %d", IPRequest, msg.Type)
	}

	newIP, ok := s.nextIP()
	if !ok {
		_, err := c.Write(HandshakeMessage{Type: NoIPError}.ToBytes())
		return nil, err
	}

	ipbuf := [128]byte{}
	prefix, _ := s.gw.Mask.Size()
	copy(ipbuf[:], append(append(newIP, byte(prefix)), []byte(s.gw.IP.To4())...))
	_, err = c.Write(HandshakeMessage{
		Type:    IPResponse,
		Payload: ipbuf,
	}.ToBytes())
	if err != nil {
		return nil, err
	}
	msg, err = getMessage(c)
	if err != nil {
		return nil, err
	}

	if msg.Type != Accept {
		return newIP, fmt.Errorf("client didn't accept the IP address")
	}

	_, err = c.Write(HandshakeMessage{
		Type:    AcknowledgeAccept,
		Payload: s.motd,
	}.ToBytes())
	return newIP, err
}

// AddClient adds a new connection session struct for the passed connection
// and returns a connSession for the connection.
func (s *Server) AddClient(ip net.IP, conn net.Conn) connSession {
	s.clientsMutex.Lock()
	pr, pw := io.Pipe()
	s.clients[string(ip)] = connSession{
		conn:       conn,
		filteredRd: pr,
		filteredWr: pw,
	}
	s.clientsMutex.Unlock()
	return s.clients[string(ip)]
}

// RemoveClient removes the connection instance from the server state and closes
// all the associated pipes and the connection itself.
func (s *Server) RemoveClient(ip net.IP) {
	s.clientsMutex.Lock()
	cs, ok := s.clients[string(ip)]
	if ok {
		if cs.filteredRd != nil {
			cs.filteredRd.Close()
		}
		if cs.filteredWr != nil {
			cs.filteredWr.Close()
		}
		if cs.conn != nil {
			cs.conn.Close()
		}
	}
	delete(s.clients, string(ip))
	s.clientsMutex.Unlock()
}

// MuxTCP method of the server reads all the generic packets
// received back from the tunneling interface and writes to
// right destination based on the destination IP address
// defined in the IP packet.
func (s *Server) MuxTCP() {
	p := make([]byte, MTU)
	for {
		n, err := s.iface.Read(p)
		if err != nil {
			fmt.Println("Interface read error: ", err)
			return
		}
		dst := p[16:20]
		cl, ok := s.clients[string(dst)]
		if !ok {
			continue
		}

		_, err = cl.filteredWr.Write(p[:n])
		if err != nil {
			fmt.Println(err)
		}
	}
}

// Close closes the server instance
func (s *Server) Close() {
	s.server.Close()
	s.iface.Close()
}

// InterfaceSession is a filter scope of the network interface
// which only reads the IP packets meant for a specific client.
// This IP based filtering happens in the server's mux construct.
type InterfaceSession struct {
	Reader io.Reader
	Writer io.Writer
}

// Write implements io.Writer method for the InterfaceSession.
// Packets sent to this are directly written to the network interface.
func (is *InterfaceSession) Write(p []byte) (int, error) {
	return is.Writer.Write(p)
}

// Read implements io.Reader method for the InterfaceSession.
// Packets read from this are filtered from the other end of the pipe
// based on the destination IP received from the remote packets.
func (is *InterfaceSession) Read(p []byte) (int, error) {
	return is.Reader.Read(p)
}
