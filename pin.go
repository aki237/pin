package main

import (
	"encoding/base64"
	"fmt"
	"net"
	"os"
	"syscall"

	"./pinlib"
)

func RunPin(config *Config, c chan os.Signal) {
	server := config.Mode == SERVER
	var err error
	var session Session
	session.Config = config
	iface := NewTUN(&session.InterfaceName)

	defer iface.Close()

	secretdec, err := base64.StdEncoding.DecodeString(session.Secret)
	if err != nil {
		fmt.Println(err)
		return
	}

	if len(secretdec) != 32 {
		fmt.Println("Error : key length mismatch, need 40 got", len(secretdec))
		return
	}

	var kcn [32]byte
	copy(kcn[:], secretdec)

	if server {
		var ipNet *net.IPNet
		var ip net.IP
		ip, ipNet, err = net.ParseCIDR(session.DHCP)
		if err != nil {
			fmt.Println(err)
			return
		}
		ipNet.IP = ip
		session.peer, err = pinlib.NewServer(session.Address, iface, ipNet, kcn)
		if err != nil {
			fmt.Println(err)
			return
		}
		err = session.SetupServer()
		if err != nil {
			fmt.Println(err)
			return
		}
	} else {
		session.peer = pinlib.NewClient(session.Address, iface, kcn)

		session.SetupClient()
	}

	go func() {
		recdsig := <-c
		switch recdsig {
		case syscall.SIGTERM, os.Interrupt:
			fmt.Println("\nReceived Ctrl-C")
		case syscall.SIGTSTP:
			fmt.Println("\nReceived Ctrl-Z. Suspend not supported.")
		}
		session.peer.Close()
		fmt.Println("Exchanger Closed...")
	}()

	err = session.peer.Start()
	if err != nil {
		fmt.Println(err)
	}

	iface.Close()

	if !server {
		fmt.Println("Stopping client.")
		session.StopClient()
		fmt.Println("Stopped client.")
	}
}
