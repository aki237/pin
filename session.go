package main

import (
	"net"

	"gitlab.com/aki237/pin/pinlib"
)

// Session struct is used to hold the session values like the local tunneling address
// and other information about the session. This also contains information about the
// server to dial or the address to listen at.
type Session struct {
	*Config
	ResolvedRemoteIP net.IP      // Contains the resolved IPv4 address of remote Peer
	RemotePort       int         // Contains the port of the server the client is connecting to
	InterfaceAddress string      // to be setup during in the hook function
	InterfaceGateway string      // to be setup during in the hook function
	peer             pinlib.Peer // to be setup before the connection initialization
}
