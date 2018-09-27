package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"pinlib"
)

var (
	version   = "0.0.0"
	buildDate = "HH:MM DD-MMM-YYYY"
)

func printVersionInfo() {
	fmt.Println("pin v" + version + " " + buildDate)
}

func main() {
	configFile := flag.String("c", "", "config file to parse")
	versionPrint := flag.Bool("v", false, "print the version info")

	flag.Usage = func() {
		printVersionInfo()
		flag.VisitAll(func(f *flag.Flag) {
			fmt.Printf("\t-%s\t%s\n", f.Name, f.Usage+f.DefValue)
		})
	}
	flag.Parse()

	if *versionPrint {
		printVersionInfo()
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
		RunPin(config, c)
	case CLIENT:
		RunPin(config, c)
	default:
		fmt.Println("How did you even make it till here?? `:|")
	}
}
