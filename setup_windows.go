// +build windows
package main

import (
	"fmt"

	"./pinlib"
)

func SetupClient(client *pinlib.Client, addr, ifaceName string) {
	client.Hook = func(ipp string, gw string) error {
		return nil
	}
}

func SetupServer(server *pinlib.Server, ifaceName, tunaddr string) error {
	fmt.Println("Not implemented...")
	return nil
}

func StopClient(addr string) {
	fmt.Println("Not implemented")
}
