package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"gitlab.com/aki237/pin/pinlib"
)

var (
	version   = "0.0.0"
	buildDate = ""
)

func printVersionInfo() {
	fmt.Println("pin v" + version + " " + buildDate)
}

func main() {
	versionPrint := flag.Bool("v", false, "print the version info")

	flag.Usage = func() {
		printVersionInfo()
		fmt.Printf("Usage:\n\tpin [options] <config>\n\n")
		fmt.Printf("Options:\n")
		flag.VisitAll(func(f *flag.Flag) {
			fmt.Printf("\t-%s\t%s\n", f.Name, fmt.Sprintf("%s [default: %v]", f.Usage, f.DefValue))
		})
	}
	flag.Parse()

	if *versionPrint {
		printVersionInfo()
		return
	}

	if len(flag.Args()) != 1 {
		flag.Usage()
		return
	}

	configFile := flag.Arg(0)

	config, err := NewConfigFromFile(configFile)
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
