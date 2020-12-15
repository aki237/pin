// Package pinlib implements client and server functionaljity for a pin based VPN service.
package pinlib

import (
	"fmt"
	"io"
	"net"
	"runtime"
)

type CallbackConn struct {
	io.ReadWriteCloser
	ip       net.IP
	Callback func()
}

// Exchanger is the main struct used to enable IP packet transfer between 2 peers
// This is the basis for functionality of both the client and server
type Exchanger struct {
	conn    io.ReadWriter
	iface   io.ReadWriter
	running bool
}

// Start method starts the IP packet exchange between the configured interface and the TCP connection
func (p *Exchanger) Start() {
	p.running = true
	go p.outgoing()
	p.incoming()
}

// incoming method reads IP data from the TCP connection, decompresses it and writes it to the configured tunneling interface.
func (p *Exchanger) incoming() {
	// the buffer size is MTU which should be the configured MTU for the tunneling interface.
	packet := make([]byte, MTU)

	// pinlib exchanger uses snappy compression which gave good results in transfer speeds.
	rd := p.conn

	for p.running {
		n, err := rd.Read(packet)
		if err != nil {
			if p.running {
				fmt.Println("Incoming_Read: ", err)
			}
			if nc, ok := p.conn.(*CallbackConn); ok {
				nc.Callback()
			}
			p.running = false
			return
		}

		// For openbsd, there is a additional tunnel header of Address
		// family of the packet to be added.
		//
		// In this case it is AF_INET (0x02)
		if runtime.GOOS == "openbsd" {
			packet = append([]byte{0, 0, 0, 2}, packet[:]...)
			n = n + 4
		}
		p.iface.Write(packet[:n])
	}
}

// outgoing method reads IP data from the configured tunneling interface, compresses it and writes it to the TCP connection
func (p *Exchanger) outgoing() {
	// the buffer size is MTU which should be the configured MTU for the tunneling interface.
	packet := make([]byte, MTU)

	// snappy compressor interface
	wr := p.conn

	for p.running {
		n, err := p.iface.Read(packet)
		if err != nil {
			fmt.Println("Outgoing_Read: ", err)
			p.running = false
			return
		}

		// In openbsd, there is a additional tunnel header to specify the protocol
		// This had to be discarded before sending it to the remote.
		if runtime.GOOS == "openbsd" {
			if len(packet) < 4 {
				packet = packet[4:]
				n = n - 4
				continue
			}
		}

		_, err = wr.Write(packet[:n])
		if err != nil {
			if p.running {
				fmt.Println("Outgoing_Write: ", err)
			}
			if nc, ok := p.conn.(*CallbackConn); ok {
				nc.Callback()
			}
			p.running = false
			return
		}
	}

	fmt.Println("Exchanger : closing the outgoing txn")
}
