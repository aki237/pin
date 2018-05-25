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
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c

		fmt.Println("\n\nTyring to exit gracefully\n\n")
		StopClient(addr)
		os.Exit(1)
	}()

	err = handler.Start()
	if err != nil {
		fmt.Println(err)
	}

	iface.Close()

	fmt.Println("\n\n\n\n\nerr\n\n\n\n\n")
}

/*
func RevertRemoteRouting(addr string) error {
	if runtime.GOOS != "linux" {
		fmt.Println("Not implemented yet")
		return nil
	}
	ta, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return err
	}

	fmt.Println(ta.IP)

	rs, err := netlink.RouteGet(ta.IP)
	if err != nil {
		return err
	}

	rs[0].Src = nil

	return netlink.RouteDel(&rs[0])
}
*/
