package main

import (
	"flag"
	"fmt"
)

func main() {
	mode := flag.Bool("s", false, "switch on server mode instear of client")
	addr := flag.String("addr", "", "(client mode) address of the server\n(server mode) local listening address")
	ifaceName := flag.String("i", "pin0", "name of the tunneling network interface")
	tunaddr := flag.String("tunaddr", "", "IP address of the tunneling network interface")
	gw := flag.String("gw", "", "(client mode only) IP address of the remote tunnel interface which acts as the routing gateway")
	flag.Parse()

	if *addr == "" {
		fmt.Println("Error::Commandline::Parse : not a valid address")
		return
	}

	if *tunaddr == "" {
		fmt.Println("Error::Commandline::Parse : no IP address for the tunnel interface is provided")
		return
	}

	if *mode && *gw != "" {
		fmt.Println("Error::Commandline::Parse : gateway is only needed in the client side")
		return
	}

	RunPin(*mode, *addr, *ifaceName, *tunaddr, *gw)
}
