package main

import (
	"fmt"

	"./pinlib"
)

func RunPin(mode bool, addr, ifaceName, tunaddr, gw string) {

	iface := NewTUN(&ifaceName)
	defer iface.Close()

	var handler pinlib.Peer
	var err error

	if !mode {
		handler, err = pinlib.NewClient(addr, 1, iface)
		if err != nil {
			fmt.Println(err)
			return
		}

		SetupClient(handler.(*pinlib.Client), addr, ifaceName, tunaddr, gw)

	} else {
		handler, err = pinlib.NewServer(addr, iface)
		if err != nil {
			fmt.Println(err)
			return
		}
		err = SetupServer(handler.(*pinlib.Server), ifaceName, tunaddr)
		if err != nil {
			fmt.Println(err)
			return
		}
	}

	err = handler.Start()
	if err != nil {
		fmt.Println(err)
		return
	}

	iface.Close()

}
