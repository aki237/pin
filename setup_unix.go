// +build darwin dragonfly freebsd openbsd netbsd !linux

package main

import (
	"errors"
	"fmt"
	"net"
	"os/exec"
	"syscall"

	"./pinlib"
	"golang.org/x/net/route"
)

func getDefaultGatewayIP() (net.IP, error) {
	routedata, err := route.FetchRIB(syscall.AF_INET, route.RIBTypeRoute, 0)
	if err != nil {
		return nil, err
	}

	msg, err := route.ParseRIB(route.RIBTypeRoute, routedata)
	if err != nil {
		return nil, err
	}
	for _, rt := range msg {
		rtmsg, ok := rt.(*route.RouteMessage)
		if !ok {
			continue
		}

		if rtmsg.Flags&0x2 == 0x2 {
			if len(rtmsg.Addrs) < 3 {
				continue
			}
			src, ok := rtmsg.Addrs[0].(*route.Inet4Addr)
			if !ok {
				continue
			}
			gw, ok := rtmsg.Addrs[1].(*route.Inet4Addr)
			if !ok {
				continue
			}

			mask, ok := rtmsg.Addrs[2].(*route.Inet4Addr)
			if !ok {
				continue
			}
			comp := string([]byte{0, 0, 0, 0})
			if string(src.IP[:]) == comp && string(mask.IP[:]) == comp {
				return gw.IP[:], nil
			}
		}
	}

	return nil, errors.New("default gateway not found")
}

func SkipRemoteRouting(addr string) error {
	gw, err := getDefaultGatewayIP()
	if err != nil {
		return err
	}
	gwString := fmt.Sprintf("%d.%d.%d.%d", gw[0], gw[1], gw[2], gw[3])
	ta, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return err
	}

	ta.IP = ta.IP.To4()

	serverIP := fmt.Sprintf("%d.%d.%d.%d", ta.IP[0], ta.IP[1], ta.IP[2], ta.IP[3])
	cmd := exec.Command("route", "add", serverIP, gwString)
	return cmd.Run()

}

func SetupAddr(ifaceName, ipp, gateway string) error {
	cmd := exec.Command("ifconfig", ifaceName, ipp, gateway)
	return cmd.Run()
}

func SetupRoutes(gw string) error {
	routeA := exec.Command("route", "add", "128.0.0.0/1", gw)
	routeB := exec.Command("route", "add", "0.0.0.0/1", gw)
	err := routeA.Run()
	if err != nil {
		return err
	}
	return routeB.Run()
}

func SetupClient(client *pinlib.Client, addr, ifaceName string) {
	client.Hook = func(ipp, gw string) error {
		err := SkipRemoteRouting(addr)
		if err != nil {
			fmt.Printf("Error while adding the blacklist route (ie., %s through the default gw): %s\n", addr, err)
		}
		err = SetupAddr(ifaceName, ipp, gw)
		if err != nil {
			return err
		}
		return SetupRoutes(gw)
	}

}

func SetupServer(server *pinlib.Server, ifaceName, tunaddr string) error {
	fmt.Println("Not Implemented")
	return nil
}

func StopClient(addr string) {
	ta, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		fmt.Println(err)
		return
	}

	ta.IP = ta.IP.To4()

	serverIP := fmt.Sprintf("%d.%d.%d.%d", ta.IP[0], ta.IP[1], ta.IP[2], ta.IP[3])
	cmd := exec.Command("route", "delete", serverIP)
	err = cmd.Run()
	if err != nil {
		fmt.Println("Route for remote server cannot be removed: ", err)
		return
	}

	return
}
