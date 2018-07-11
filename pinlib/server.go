package pinlib

import (
	"fmt"
	"io"
	"math/rand"
	"net"
	"sync"
)

// Server struct contains all fields for exchanging packets to the client through a TCP connection
type Server struct {
	gw      *net.IPNet
	server  net.Listener
	iface   io.ReadWriter
	running bool
	secret  [32]byte
	close   chan bool
}

// NewServer method is used to create a new server struct with a given listening address
func NewServer(addr string, iface io.ReadWriter, gw *net.IPNet, secret [32]byte) (*Server, error) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	return &Server{server: ln, iface: iface, gw: gw, running: false, secret: secret, close: make(chan bool)}, nil
}

type NotifierConn struct {
	io.ReadWriteCloser
	ip   string
	comm chan string
	wg   *sync.WaitGroup
}

func (conn *NotifierConn) Notify() {
	conn.ReadWriteCloser.Close()
	conn.comm <- conn.ip
	conn.wg.Done()
}

func (s *Server) nextIP(lastIP net.IP) (net.IP, bool) {
	for i := len(lastIP) - 1; i >= 0; i-- {
		lastIP[i]++
		if lastIP[i] != 0 {
			break
		}
	}
	if !s.gw.Contains(lastIP) {
		return nil, false
	}
	return lastIP.To4(), true
}

func foundInMap(k string, dict map[string]io.WriteCloser) bool {
	for key, _ := range dict {
		if key == k {
			return true
		}
	}
	return false
}

// Start method accepts TCP connections from a client and starts the packet exchange from the local tunneling interface to the remote client
// This also makes Server struct to satisfy the pinlib.Peer interface.
func (s *Server) Start() error {
	wg := &sync.WaitGroup{}

	mux := &ifaceMux{conn: make(map[string]io.WriteCloser, 0), iface: s.iface, sig: make(chan string)}

	s.running = true

	// TODO : profile the runtime for multiplexing
	go mux.Mux()
	go mux.Mux()
	go mux.Mux()
	go mux.Mux()
	go mux.Mux()
	go mux.Mux()
	go mux.cleanup()

	lastIP := make(net.IP, 4)
	copy(lastIP, s.gw.IP.To4())

	fmt.Println(lastIP)
	go func() {
		for s.running {
			cx, err := s.server.Accept()
			if err != nil {
				fmt.Println(err)
				continue
			}

			seed := rand.Intn(255)
			cx.Write([]byte{byte(seed)})

			conn := NewCryptoConn(cx, s.secret, int64(seed))

			hreq := make([]byte, 5)
			_, err = conn.Read(hreq)
			if err != nil {
				fmt.Println(err)
				continue
			}

			if string(hreq) != "IPPLS" {
				fmt.Println("Discarding connection due to wrong handshake request from: ", conn.RemoteAddr(), string(hreq))
				continue
			}

			var available bool

			lastIP, available = s.nextIP(lastIP)
			if !available {
				xx := make(net.IP, 4)
				copy(xx, s.gw.IP.To4())
				var found bool
				var avail bool = false
				for {
					xx, found = s.nextIP(xx)
					if !found {
						break
					}
					if !foundInMap(string(xx), mux.conn) {
						lastIP = xx.To4()
						avail = true
						break
					}
				}
				if !avail {
					conn.Write([]byte{0})
					continue
				}
			}

			prefix, _ := s.gw.Mask.Size()
			hsd := append([]byte(lastIP), byte(prefix))
			hsd = append(hsd, []byte(s.gw.IP.To4())...)
			_, err = conn.Write(hsd)
			if err != nil {
				fmt.Println(err)
				continue
			}

			p := make([]byte, 1)

			_, err = conn.Read(p)

			if err != nil {
				fmt.Println(err)
				continue
			}

			if p[0] != 1 {
				lastIP[3]--
				fmt.Println("client wasn't happy")
				continue
			}

			fmt.Println("Negotiated addr : ", lastIP)

			pr, pw := io.Pipe()

			mux.conn[string(lastIP)] = pw

			ex := &Exchanger{conn: &NotifierConn{ReadWriteCloser: conn, ip: string(lastIP), comm: mux.sig, wg: wg}, iface: &ifaceClient{pr: pr, wr: s.iface, addr: p}}
			wg.Add(1)
			go ex.Start()
		}
	}()

	<-s.close
	fmt.Println("Closing existing connections...")
	//mux.Close()
	fmt.Println("Closed all the muxes")
	return nil
}

func (s *Server) Close() {
	s.running = false
	s.close <- true
}

type ifaceClient struct {
	addr []byte
	pr   io.Reader // pipe reader end
	wr   io.Writer // iface fd itself
}

// To client
func (i *ifaceClient) Read(p []byte) (int, error) {
	return i.pr.Read(p)
}

// From client
func (i *ifaceClient) Write(p []byte) (int, error) {
	return i.wr.Write(p)
}

type ifaceMux struct {
	conn  map[string]io.WriteCloser // pipe's writing end
	iface io.Reader
	sig   chan string
}

// Sendback muxing
func (m *ifaceMux) Mux() {
	p := make([]byte, MTU)
	for {
		n, _ := m.iface.Read(p)
		dst := p[16:20]
		cl, ok := m.conn[string(dst)]
		if !ok {
			continue
		}

		_, err := cl.Write(p[:n])
		if err != nil {
			fmt.Println(err)
		}
	}
}

func (m *ifaceMux) cleanup() {
	for {
		clientIp := <-m.sig
		fmt.Println("Removed client : ", []byte(clientIp))
		delete(m.conn, clientIp)
	}
}

func (i *ifaceMux) Close() {
	for _, val := range i.conn {
		val.Close()
	}

	close(i.sig)
}
