// Package pinlib implements client and server functionaljity for a pin based VPN service.
package pinlib

import (
	"fmt"
	"io"
	"runtime"

	"github.com/golang/snappy"
)

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
	rd := snappy.NewReader(p.conn)

	for p.running {
		n, err := rd.Read(packet)
		if err != nil {
			if p.running {
				fmt.Println("Incoming_Read: ", err)
			}
			if nc, ok := p.conn.(*NotifierConn); ok {
				nc.Notify()
			}
			p.running = false
			return
		}

		// For openbsd, there is a additional tunnel header to be added.
		// In this case, the tunnel being used is PPTP so the byte array 0,0,0,2 is used.
		// See man 4 pppx and /usr/include/net/pipex.h (PIPEX_PROTO_PPTP)
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
	wr := snappy.NewWriter(p.conn)

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
			packet = packet[4:]
			n = n - 4
		}

		_, err = wr.Write(packet[:n])
		if err != nil {
			if p.running {
				fmt.Println("Outgoing_Write: ", err)
			}
			if nc, ok := p.conn.(*NotifierConn); ok {
				nc.Notify()
			}
			p.running = false
			return
		}
	}

	fmt.Println("Exchanger : closing the outgoing txn")
}
