// +build darwin
package main

import (
	"fmt"
	"log"

	"github.com/songgao/water"
)

func NewTUN(name *string) *water.Interface {
	cfg := water.Config{DeviceType: water.TUN}

	iface, err := water.New(cfg)
	if err != nil {
		log.Fatal(err)
	}
	*name = iface.Name()
	return iface
}

func StopClient(addr string) {
	fmt.Println("Not implemented")
}
