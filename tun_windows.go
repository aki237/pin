// +build windows
package main

import (
	"log"

	"github.com/songgao/water"
)

func NewTUN(name *string) *water.Interface {
	cfg := water.Config{DeviceType: water.TUN}
	cfg.ComponentID = *name
	cfg.Network = "10.0.0.2/24"

	iface, err := water.New(cfg)
	if err != nil {
		log.Fatal(err)
	}
	*name = iface.Name()
	return iface
}
