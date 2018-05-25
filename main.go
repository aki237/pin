package main

import (
	"flag"
	"fmt"

	"./pinlib"
)

func main() {
	// server only options
	mode := flag.Bool("s", false, "switch on server mode instead of client")
	dhcp := flag.String("dhcp", "", "(server mode) info for dhcp server")

	// common options
	ifaceName := flag.String("i", "pin0", "name of the tunneling network interface")
	mtu := flag.Int("mtu", 1500, "specify MTU of the tunneling Device")
	pub := flag.String("pub", "enc.pem", "Pubic key")
	key := flag.String("priv", "enc.key", "Private key")
	addr := flag.String("addr", "", "IP address of the tunneling network interface")

	// client options

	flag.Parse()

	pinlib.MTU = *mtu

	if *dhcp == "" && *mode {
		fmt.Println("Error::Commandline::Parse : no IP address for the tunnel interface is provided")
		return
	}

	if !*mode && *addr == "" {
		fmt.Println("Error::Commandline::Parse : no remote server specified")
		return
	}

	RunPin(*mode, *addr, *ifaceName, *dhcp, *pub, *key)
}
