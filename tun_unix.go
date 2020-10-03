// +build darwin dragonfly freebsd openbsd netbsd !linux

package main

import (
	"log"

	"gitlab.com/sbioa1234/water"
)

// NewTUN is used to create a TUN device for macOS, FreeBSD, DragonflyBSD,
// OpenBSD and NetBSD.
func NewTUN(name *string) *water.Interface {
	cfg := water.Config{DeviceType: water.TUN}

	iface, err := water.New(cfg)
	if err != nil {
		log.Fatal(err)
	}
	*name = iface.Name()
	return iface
}
