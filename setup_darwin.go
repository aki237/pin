// +build darwin
package main

import (
	"fmt"

	"./pinlib"
)

func SetupClient(client *pinlib.Client, addr, ifaceName string) {
	client.Hook = func(ipp string, gw string) error {
		fmt.Println("Not implemented...", ipp, gw)
		return nil
	}
}

func SetupServer(server *pinlib.Server, ifaceName, tunaddr string) error {
	fmt.Println("Not implemented...")
	return nil
}
