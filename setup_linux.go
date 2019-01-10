// +build linux

package main

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"strings"

	"gitlab.com/aki237/pin/pinlib"

	"github.com/vishvananda/netlink"
)

// This file mainly contains helper functions for client and server side setup after the
// handshake connection is established

func getDefaultRoutes(addr string) ([]netlink.Route, error) {
	ipaddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return nil, err
	}
	return netlink.RouteGet(ipaddr.IP)
}

func getDefaultGateway(addr string) (net.IP, error) {
	routes, err := getDefaultRoutes(addr)
	if err != nil {
		return nil, err
	}
	if len(routes) == 0 {
		return nil, errors.New("no route to host")
	}
	return routes[0].Gw, nil
}

func getDefaultLinkDevIndex() (int, error) {
	routes, err := getDefaultRoutes("8.8.8.8:53")
	if err != nil {
		return -1, err
	}
	if len(routes) == 0 {
		return -1, errors.New("no route to host")
	}

	return routes[0].LinkIndex, nil
}

func SkipRemoteRouting(addr string) error {
	gw, err := getDefaultGateway(addr)
	if err != nil {
		return err
	}

	ta, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return err
	}

	err = netlink.RouteAdd(&netlink.Route{
		Dst: &net.IPNet{
			IP:   ta.IP,
			Mask: net.IPv4Mask(255, 255, 255, 255),
		},
		Gw: gw,
	})

	if err != nil {
		if !os.IsExist(err) {
			return err
		}
	}

	return nil
}

func SetupRoutes(remotegw string) error {
	gw, err := net.ResolveIPAddr("ip4", remotegw)
	if err != nil {
		return err
	}
	err = netlink.RouteAdd(&netlink.Route{
		Dst: &net.IPNet{
			IP:   []byte{0, 0, 0, 0},
			Mask: net.IPv4Mask(128, 0, 0, 0),
		},
		Gw: gw.IP,
	})

	if err != nil {
		return err
	}

	return netlink.RouteAdd(&netlink.Route{
		Dst: &net.IPNet{
			IP:   []byte{128, 0, 0, 0},
			Mask: net.IPv4Mask(128, 0, 0, 0),
		},
		Gw: gw.IP,
	})
}

func SetupAddr(ifaceName string, ifaceAddr string, remotegw string) error {
	// get the link holder
	link, err := netlink.LinkByName(ifaceName)
	if err != nil {
		return err
	}

	addr, err := netlink.ParseAddr(ifaceAddr)
	if err != nil {
		return err
	}

	if remotegw != "" {

		ipaddr, err := net.ResolveIPAddr("ip4", remotegw)
		if err != nil {
			return err
		}
		addr.Peer = &net.IPNet{IP: ipaddr.IP, Mask: net.IPv4Mask(255, 255, 255, 255)}

	}
	return netlink.AddrAdd(link, addr)
}

func SetupLink(ifaceName string) error {
	// get the link holder
	link, err := netlink.LinkByName(ifaceName)
	if err != nil {
		return err
	}

	// set the mtu
	err = netlink.LinkSetMTU(link, pinlib.MTU)
	if err != nil {
		return err
	}

	// activate it
	return netlink.LinkSetUp(link)

}

func SetupIPTables(ifaceName string) error {
	// iptables -F
	cmd, err := findExecutablePath("iptables")
	if err != nil {
		return fmt.Errorf("probably iptables command is missing from your system (?) or not found in the $PATH, make sure it is available : %s", err)
	}

	ix, err := getDefaultLinkDevIndex()
	if err != nil {
		return err
	}

	link, err := netlink.LinkByIndex(ix)
	if err != nil {
		return err
	}

	cmds := [][]string{
		{"-F"},              // Flush any old rules
		{"-F", "-t", "nat"}, // Flush the same for the NAT table

		{"-I", "FORWARD", "-i", ifaceName, "-j", "ACCEPT"},                              // Accept all input packets from "interface" in the FORWARD chain
		{"-I", "FORWARD", "-o", ifaceName, "-j", "ACCEPT"},                              // Accept all output packets from "interface" in the FORWARD chain
		{"-I", "INPUT", "-i", ifaceName, "-j", "ACCEPT"},                                // Accept all output packets from "interface" in the INPUT chain
		{"-t", "nat", "-I", "POSTROUTING", "-o", link.Attrs().Name, "-j", "MASQUERADE"}, // It says what it does ;)
	}

	for _, cx := range cmds {
		log.Println("running command : ", strings.Join(append([]string{cmd}, cx...), " "))
		err := exec.Command(cmd, cx...).Start()
		if err != nil {
			return err
		}
	}

	return err
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
			return err
		}

		err = SetupLink(s.InterfaceName)
		if err != nil {
			return err
		}

		var ip net.IP

		ip, s.InterfaceAddress, err = net.ParseCIDR(ipp)
		if err != nil {
			return err
		}

		s.InterfaceAddress.IP = ip.To4()

		s.InterfaceGateway = gw

		err = SetupAddr(s.InterfaceName, ipp, gw)
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
	err := SetupLink(s.InterfaceName)
	if err != nil {
		return err
	}

	err = SetupAddr(s.InterfaceName, s.DHCP, "")
	if err != nil {
		return err
	}

	return SetupIPTables(s.InterfaceName)
}

func (s *Session) StopClient() {
	netlink.RouteDel(&netlink.Route{
		Dst: &net.IPNet{
			IP:   s.ResolvedRemote,
			Mask: net.IPv4Mask(255, 255, 255, 255),
		},
		Gw: s.DefaultGateway,
	})
	err := s.RevertDNS()
	if err != nil {
		fmt.Println("\n * Unable to revert the DNS settings : ", err)
		fmt.Println(" * Add the following to the /etc/resolv.conf file (till the line with the '# %%') *")
		fmt.Println(s.ocresolv)
		fmt.Println("# %%\n")
	}
}
