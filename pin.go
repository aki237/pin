package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func runPin(config *Config) {
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGTSTP)
	session, err := config.GetSession()
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
		fmt.Print("\r Closing stuff")
	}()

	if session.Mode != SERVER {
		defer session.StopClient()
	}

	err = session.peer.Start()
	if err != nil {
		fmt.Println(err)
	}
}
