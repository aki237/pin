package pinlib

import (
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
)

// Client struct contains all fields for exchanging packets to the server through a TCP connection
type Client struct {
	// Unexported
	iface  io.ReadWriteCloser
	secret [32]byte
	conn   net.Conn

	// Exported
	Protocol string
	RemoteIP net.IP
	Port     int

	// TODO: change this to get net.IP and net.IPNet instead
	Hook func(ip, gw string) error // Hook is a function that runs immediately after the TCP connection is made
}

// NewClient is used to create a new client which makes a connection to the remote pin.
func NewClient(ip net.IP, port int, proto string, iface io.ReadWriteCloser, secret [32]byte) *Client {
	// if number of connections is 0 it is pointless to run this VPN

	return &Client{
		iface:    iface,
		RemoteIP: ip, Port: port, Protocol: proto,
		secret: secret,
		Hook:   func(ip, gw string) error { return nil },
	}
}

// Start method makes TCP connections and starts the packet exchange from the local tunneling interface to the remote interface.
// This also makes Client struct to satisfy the pinlib.Peer interface.
func (c *Client) Start() error {
	// wait group to wait for all go routines to complete
	switch c.Protocol {
	case "tcp":
		return c.StartTCP()
	case "udp":
		return fmt.Errorf("Not implemented")
	default:
		return fmt.Errorf("Unsupported protocol")
	}
}

// StartTCP starts the packet exchange with remote.
// Here the IP address negotiation happens for the tunnel interface.
func (c *Client) StartTCP() error {
	cx, err := net.DialTCP("tcp", nil, &net.TCPAddr{IP: c.RemoteIP, Port: c.Port})
	if err != nil {
		return err
	}
	conn := NewCryptoConn(cx, c.secret)
	c.conn = conn

	ip, ipnet, motd, err := c.HandshakeTCP()
	if err != nil {
		return err
	}
	prefix, _ := ipnet.Mask.Size()

	// this is where the hook function is run.
	// Generally for a pinlib based VPN program, this Hook function should be configured with IP routing and device setup
	err = c.Hook(fmt.Sprintf("%s/%d", ip.String(), prefix), ipnet.IP.String())
	if err != nil {
		return err
	}
	sep := strings.Repeat("-", 18)
	fmt.Printf("Message of the day\n%s\n%s\n%s\n", sep, motd, sep)

	ex := &Exchanger{
		conn: &CallbackConn{
			ReadWriteCloser: conn,
			ip:              ip,
			Callback:        c.Close,
		},
		iface: c.iface,
	}
	ex.Start()
	return nil
}

func (c *Client) HandshakeTCP() (net.IP, *net.IPNet, []byte, error) {
	_mkmsg := func(t MessageType) *HandshakeMessage {
		return &HandshakeMessage{
			Type: t,
		}
	}

	if _, err := c.conn.Write(_mkmsg(IPRequest).ToBytes()); err != nil {
		return nil, nil, nil, err
	}

	msg, err := getMessage(c.conn)
	if err != nil {
		return nil, nil, nil, err
	}

	if msg.Type != IPResponse {
		return nil, nil, nil, errors.New("Server did not return a IP")
	}

	ip := msg.Payload[:4]
	ipnet := &net.IPNet{
		IP:   msg.Payload[5:9],
		Mask: net.CIDRMask(int(msg.Payload[4]), 0),
	}

	if _, err := c.conn.Write(_mkmsg(Accept).ToBytes()); err != nil {
		return nil, nil, nil, err
	}

	msg, err = getMessage(c.conn)
	if err != nil {
		return nil, nil, nil, err
	}
	if msg.Type != AcknowledgeAccept {
		return nil, nil, nil, fmt.Errorf("expected Ack, got: %d", msg.Type)
	}
	motd := []byte{}

	for i := range msg.Payload {
		if msg.Payload[i] == 0 {
			motd = msg.Payload[:i]
		}
	}

	return ip, ipnet, motd, nil
}

// Close deinitializes the client session by closing
// the connection and the tunneling interface
func (c *Client) Close() {
	c.conn.Close()
	c.iface.Close()
}
