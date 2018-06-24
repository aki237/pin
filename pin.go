package main

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"net"
	"os"
	"strings"
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

func RunPin(server bool, addr, ifaceName, tunaddr, secret string, px *Daemon, c chan os.Signal) {
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

	if px != nil {
		px.peer = handler
		px.isConfigured = true
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

type Daemon struct {
	peer         pinlib.Peer
	isConfigured bool
	ln           net.Listener
	running      bool
	controller   chan os.Signal
}

func NewDaemon() *Daemon {
	return &Daemon{controller: make(chan os.Signal, 2)}
}

func (d *Daemon) RunSocket(pidfile string) {
	ln, err := net.Listen("unix", pidfile)
	if err != nil {
		fmt.Println(err)
		return
	}
	d.ln = ln

	defer ln.Close()

	d.running = true

	for d.running {
		conn, err := d.ln.Accept()
		if d.running {

			if err != nil {
				fmt.Println("RunSocket : ", err)
				continue
			}
		}
		go d.Handle(conn)
	}
}

func (d *Daemon) RunPin(server bool, addr, ifaceName, tunaddr, secret string) {
	if server {
		d.peer = new(pinlib.Server)
	} else {
		d.peer = new(pinlib.Client)
	}

	RunPin(server, addr, ifaceName, tunaddr, secret, d, d.controller)
	d.isConfigured = false

}

func (d *Daemon) Handle(c net.Conn) {
	rd := bufio.NewReader(c)

	for {
		line, err := rd.ReadString('\n')
		if err != nil {
			fmt.Println("RunSocket : Handle : ", err)
			return
		}

		line = strings.TrimSpace(line)
		keys := strings.Split(line, " ")
		switch keys[0] {
		case "connect":
			if len(keys) != 4 {
				fmt.Fprintf(c, "%d %s\n", ERRWARG, "wrong number of arguments")
				break
			}

			if d.isConfigured {
				fmt.Fprintf(c, "%d %s\n", ERRRUNNING, "already configured and running")
				break
			}

			go d.RunPin(false, keys[1], keys[2], "", keys[3])

			fmt.Fprintf(c, "%d %s\n", ERRSUCCESS, "success")
		case "listen":
			if len(keys) != 5 {
				fmt.Fprintf(c, "%d %s\n", ERRWARG, "wrong number of arguments")
				break
			}
			if d.isConfigured {
				fmt.Fprintf(c, "%d %s\n", ERRRUNNING, "already configured and running")
				break
			}

			go d.RunPin(true, keys[1], keys[2], keys[3], keys[4])

			fmt.Fprintf(c, "%d %s\n", ERRSUCCESS, "success")

		case "stop":
			if !d.isConfigured {
				fmt.Fprintf(c, "%d %s\n", ERRNOTRUNNING, "not configured")
				break
			}
			d.controller <- os.Interrupt
			fmt.Fprintf(c, "%d %s\n", ERRSUCCESS, "success")
		case "stat":
			if !d.isConfigured {
				fmt.Fprintf(c, "%d %s\n", ERRNOTRUNNING, "not configured")
				break
			}

			client, ok := d.peer.(*pinlib.Client)
			if !ok {
				fmt.Fprintf(c, "%d %s\n", ERRNOTIMPL, "not implemented")
				break
			}

			txn := client.GetTxnStat()

			fmt.Fprintf(c, "%d %d %d\n", ERRSUCCESS, txn.In, txn.Out)
		case "exit":
			fmt.Fprintf(c, "%d %s\n", ERRSUCCESS, "success")
			d.running = false
			d.ln.Close()
		}
	}
}
