package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"./pinlib"
)

var (
	version = "0.0.2"
)

func main() {
	configFile := flag.String("c", "", "config file to parse")
	versionPrint := flag.Bool("v", false, "print the version info")

	flag.Parse()

	if *versionPrint {
		fmt.Println("pin version v" + version)
		return
	}

	if *configFile == "" {
		flag.Usage()
		return
	}

	config, err := NewConfigFromFile(*configFile)
	if err != nil {
		fmt.Println(err)
		return
	}

	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGTSTP)

	pinlib.MTU = config.MTU
	switch config.Mode {
	case SERVER:
		RunPin(true, config.Address, config.InterfaceName, config.DHCP, config.Secret, c)
	case CLIENT:
		RunPin(false, config.Address, config.InterfaceName, config.DHCP, config.Secret, c)
	default:
		fmt.Println("How did you even make it till here?? `:|")
	}
}
