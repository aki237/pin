// +build darwin dragonfly freebsd openbsd netbsd !linux

package main

import (
	"errors"
	"fmt"
	"net"
	"os/exec"
	"runtime"
	"syscall"

	"./pinlib"
	"golang.org/x/net/route"
)

var tunname string

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
	tunname = ifaceName
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

func (s *Session) SetupClient() {
	client, ok := s.peer.(*pinlib.Client)
	if !ok {
		return
	}
	client.Hook = func(ipp, gw string) error {
		taddr, err := net.ResolveTCPAddr("tcp", s.Address)
		if err != nil {
			return err
		}
		s.ResolvedRemote = taddr.IP.To4()

		s.DefaultGateway, err = getDefaultGateway(s.Address)
		if err != nil {
			return err
		}
		err = SkipRemoteRouting(s.Address)
		if err != nil {
			fmt.Printf("Error while adding the blacklist route (ie., %s through the default gw): %s\n", s.Address, err)
		}

		_, s.InterfaceAddress, err = net.ParseCIDR(ipp)
		if err != nil {
			return err
		}

		s.InterfaceGateway = gw

		err = SetupAddr(s.InterfaceName, s.InterfaceAddress.String(), s.InterfaceGateway)
		if err != nil {
			return err
		}

		err = SetupRoutes(gw)
		if err != nil {
			return err
		}

		return s.SetupDNS()
	}

}

func (s *Session) SetupServer() error {
	fmt.Println("Not Implemented")
	return nil
}

func (s *Session) StopClient() {

	serverIP := fmt.Sprintf("%d.%d.%d.%d", s.ResolvedRemote[0], s.ResolvedRemote[1], s.ResolvedRemote[2], s.ResolvedRemote[3])
	cmd := exec.Command("route", "delete", serverIP)
	err = cmd.Run()
	if err != nil {
		fmt.Println("Route for remote server cannot be removed: ", err)
		return
	}

	if runtime.GOOS != "dragonfly" {
		cmd := exec.Command("ifconfig", tunname, "destroy")
		err = cmd.Run()
		if err != nil {
			fmt.Println("Tunnel interface cannot be removed: ", err)
			return
		}

		fmt.Println("Interface deleted:", tunname)
	}

	err := s.RevertDNS()
	if err != nil {
		fmt.Println(" * Unable to revert the DNS settings : ", err)
		fmt.Println(" * Add the following to the /etc/resolv.conf file (till the line with the '# %%') *")
		fmt.Println(s.ocresolv)
		fmt.Println("# %%")
	}

	return
}
