// +build linux
package main

import (
	"log"

	"gitlab.com/sbioa1234/water"
)

func NewTUN(name *string) *water.Interface {
	cfg := water.Config{DeviceType: water.TUN}
	cfg.Name = *name
	cfg.MultiQueue = true

	iface, err := water.New(cfg)
	if err != nil {
		log.Fatal(err)
	}
	*name = iface.Name()
	return iface
}
