package main

import (
	"encoding/base64"
	"fmt"
	"net"
	"os"
	"syscall"

	"./pinlib"
)

const (
	ERRSUCCESS    = 0
	ERRRUNNING    = 3
	ERRWARG       = 4
	ERRNOTRUNNING = 5
	ERRNOTIMPL    = 6
)

func RunPin(server bool, addr, ifaceName, tunaddr, secret string, c chan os.Signal) {
	iface := NewTUN(&ifaceName)
	defer iface.Close()

	var handler pinlib.Peer
	var err error

	secretdec, err := base64.StdEncoding.DecodeString(secret)
	if err != nil {
		fmt.Println(err)
		return
	}

	if len(secretdec) != 40 {
		fmt.Println("Error : key length mismatch, need 40 got", len(secretdec))
		return
	}

	var kcn [40]byte
	copy(kcn[:], secretdec)

	if server {
		var ipNet *net.IPNet
		var ip net.IP
		ip, ipNet, err = net.ParseCIDR(tunaddr)
		if err != nil {
			fmt.Println(err)
			return
		}
		ipNet.IP = ip
		handler, err = pinlib.NewServer(addr, iface, ipNet, kcn)
		if err != nil {
			fmt.Println(err)
			return
		}
		err = SetupServer(handler.(*pinlib.Server), ifaceName, tunaddr)
		if err != nil {
			fmt.Println(err)
			return
		}
	} else {
		handler = pinlib.NewClient(addr, iface, kcn)

		SetupClient(handler.(*pinlib.Client), addr, ifaceName)
	}

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

	if !server {
		StopClient(addr)
	}
}
