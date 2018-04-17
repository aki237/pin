// Package pinlib implements client and server functionaljity for a pin based VPN service.
package pinlib

import (
	"fmt"
	"net"
	"sync"

	"github.com/golang/snappy"
	"github.com/songgao/water"
)

// Exchanger is the main struct used to enable IP packet transfer between 2 peers
// This is the basis for functionality of both the client and server
type Exchanger struct {
	conn    net.Conn
	iface   *water.Interface
	running bool
}

// Start method starts the IP packet exchange between the configured interface and the TCP connection
func (p *Exchanger) Start(wg *sync.WaitGroup) {
	p.running = true
	go p.outgoing()
	p.incoming()
	wg.Done()
}

// incoming method reads IP data from the TCP connection, decompresses it and writes it to the configured tunneling interface.
func (p *Exchanger) incoming() {
	// the buffer size is 1500 which should be the configured MTU for the tunneling interface.
	packet := make([]byte, 1500)

	// pinlib exchanger uses snappy compression which gave good results in transfer speeds.
	rd := snappy.NewReader(p.conn)

	for p.running {
		n, err := rd.Read(packet)
		if err != nil {
			fmt.Println("Incoming_Read: ", err)
			p.running = false
			return
		}

		p.iface.Write(packet[:n])
	}

}

// outgoing method reads IP data from the configured tunneling interface, compresses it and writes it to the TCP connection
func (p *Exchanger) outgoing() {
	// the buffer size is 1500 which should be the configured MTU for the tunneling interface.
	packet := make([]byte, 1500)

	// snappy compressor interface
	wr := snappy.NewWriter(p.conn)

	for p.running {
		n, err := p.iface.Read(packet)
		if err != nil {
			fmt.Println("Outgoing_Read: ", err)
			p.running = false
			return
		}

		_, err = wr.Write(packet[:n])
		if err != nil {
			fmt.Println("Outgoing_Write: ", err)
			p.running = false
			return
		}
	}
}
