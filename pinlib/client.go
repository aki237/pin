package pinlib

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log"
	"sync"
)

// Client struct contains all fields for exchanging packets to the server through a TCP connection
type Client struct {
	// Unexported
	iface io.ReadWriter // handler for the tunneling interface
	pub   string
	key   string
	// Exported
	Remote string                    // Remote is the IP:PORT combination of the remote pin
	Hook   func(ip, gw string) error // Hook is a function that runs immediately after the TCP connection is made
	close  chan bool
}

// NewClient is used to create a new client which makes a connection to the remote pin.
func NewClient(remote string, iface io.ReadWriter, pem, priv string) *Client {
	// if number of connections is 0 it is pointless to run this VPN

	return &Client{iface: iface, Remote: remote, pub: pem, key: priv, Hook: func(ip, gw string) error { return nil }, close: make(chan bool)}
}

// Start method makes TCP connections and starts the packet exchange from the local tunneling interface to the remote interface.
// This also makes Client struct to satisfy the pinlib.Peer interface.
func (c *Client) Start() error {
	// wait group to wait for all go routines to complete
	wg := &sync.WaitGroup{}

	ipp := make([]byte, 9)

	cert, err := tls.LoadX509KeyPair(c.pub, c.key)
	if err != nil {
		log.Fatalf("client: loadkeys: %s", err)
	}
	config := tls.Config{Certificates: []tls.Certificate{cert}}
	config.InsecureSkipVerify = true

	conn, err := tls.Dial("tcp", c.Remote, &config)
	if err != nil {
		return err
	}

	n, err := conn.Read(ipp)
	if err != nil {
		return err
	}

	if n == 1 && ipp[0] == 0 {
		return errors.New("no IPs available on the server")
	}

	if n != 9 {
		return errors.New("invalid handshake")
	}

	conn.Write([]byte{1})

	fmt.Println("Connection Successful... IP Lease done : ", ipp)

	cc := &CounterConn{conn: conn}

	ex := &Exchanger{conn: cc, iface: c.iface}

	go func() {
		for !<-c.close {
		}
		ex.running = false
		conn.Close()
		wg.Done()
	}()

	// this is where the hook function is run.
	// Generally for a pinlib based VPN program, this Hook function should be configured with IP routing and device setup
	err = c.Hook(fmt.Sprintf("%d.%d.%d.%d/%d", ipp[0], ipp[1], ipp[2], ipp[3], ipp[4]),
		fmt.Sprintf("%d.%d.%d.%d", ipp[5], ipp[6], ipp[7], ipp[8]))
	if err != nil {
		return err
	}

	wg.Add(1)
	go ex.Start()

	wg.Wait()

	conn.Close()

	return nil
}

func (c *Client) Close() {
	c.close <- true
}

type CounterConn struct {
	conn              io.ReadWriter
	BytesIn, BytesOut uint64 // transfer numbers
}

func (cc *CounterConn) Read(p []byte) (int, error) {
	n, err := cc.conn.Read(p)
	cc.BytesIn += uint64(n)
	return n, err
}

func (cc *CounterConn) Write(p []byte) (int, error) {
	cc.BytesOut += uint64(len(p))
	return cc.conn.Write(p)
}
