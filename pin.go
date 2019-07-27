package main

import (
	"encoding/base64"
	"fmt"
	"net"
	"os"
	"syscall"

	"gitlab.com/aki237/pin/pinlib"
)

func RunPin(config *Config, c chan os.Signal) {

	session, err := GetSessionForConfig(config)
	if err != nil {
		fmt.Println(err)
		return
	}

	go func() {
		recdsig := <-c
		switch recdsig {
		case syscall.SIGTSTP:
			return
		}
		session.peer.Close()
		fmt.Print("\r")
	}()

	err = session.peer.Start()
	if err != nil {
		fmt.Println(err)
	}

	if session.Mode != SERVER {
		session.StopClient()
	}
}

func GetSessionForConfig(config *Config) (*Session, error) {
	server := config.Mode == SERVER
	var err error
	var session *Session = &Session{}
	session.Config = config
	iface := NewTUN(&session.InterfaceName)

	secretdec, err := base64.StdEncoding.DecodeString(session.Secret)
	if err != nil {
		return nil, err
	}

	if len(secretdec) != 32 {
		return nil, fmt.Errorf("Error : key length mismatch, need 40 got %d", len(secretdec))
	}

	var kcn [32]byte
	copy(kcn[:], secretdec)

	if server {
		var ipNet *net.IPNet
		var ip net.IP
		ip, ipNet, err = net.ParseCIDR(session.DHCP)
		if err != nil {
			return nil, err
		}
		ipNet.IP = ip
		session.peer, err = pinlib.NewServer(session.Address, iface, ipNet, kcn)
		if err != nil {
			return nil, err
		}
		err = session.SetupServer()
		if err != nil {
			return nil, err
		}
	} else {
		session.peer = pinlib.NewClient(session.Address, iface, kcn)

		session.SetupClient()
	}
	return session, nil
}
