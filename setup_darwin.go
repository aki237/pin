// +build darwin
package main

import (
	"fmt"

	"./pinlib"
)

func SetupClient(client *pinlib.Client, addr, ifaceName, tunaddr, gw string) {
	client.Hook = func() error {
		fmt.Println("Not implemented...")
		return nil
	}
}

func SetupServer(server *pinlib.Server, ifaceName, tunaddr string) error {
	fmt.Println("Not implemented...")
	return nil
}
