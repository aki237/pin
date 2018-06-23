package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"./pinlib"
)

func RunPin(mode bool, addr, ifaceName, tunaddr, pub, key string) {
	iface := NewTUN(&ifaceName)
	defer iface.Close()

	var handler pinlib.Peer
	var err error

	if !mode {
		handler = pinlib.NewClient(addr, iface, pub, key)

		SetupClient(handler.(*pinlib.Client), addr, ifaceName)

	} else {
		var ipNet *net.IPNet
		var ip net.IP
		ip, ipNet, err = net.ParseCIDR(tunaddr)
		if err != nil {
			fmt.Println(err)
			return
		}
		ipNet.IP = ip
		handler, err = pinlib.NewServer(addr, iface, ipNet, pub, key)
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

	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGTSTP)
	go func() {
		recdsig := <-c
		switch recdsig {
		case syscall.SIGTERM, os.Interrupt:
			fmt.Println("\nReceived Ctrl-C")
		case syscall.SIGTSTP:
			fmt.Println("\nReceived Ctrl-Z. Suspend not supported.")
		}
		handler.Close()
		fmt.Println("Exchanger Closed...")
	}()

	err = handler.Start()
	if err != nil {
		fmt.Println(err)
	}

	iface.Close()
	StopClient(addr)
}
