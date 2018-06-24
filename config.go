package main

import (
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
)

type RunMode int

const (
	SERVER RunMode = iota
	CLIENT
	DAEMON
)

type Config struct {
	Mode          RunMode
	Address       string
	MTU           int
	InterfaceName string
	DHCP          string
	Secret        string
	PidFile       string
}

func NewConfigFromFile(filename string) (*Config, error) {
	contents, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(contents), "\n")

	config := &Config{}

	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#") {
			continue
		}
		vals := strings.SplitN(line, ":", 2)
		if len(vals) != 2 {
			return nil, fmt.Errorf("Config parse error : error in line %d. Expected a key palue pair separated by ':'", i)
		}
		key := strings.TrimSpace(vals[0])
		val := strings.TrimSpace(vals[1])

		switch key {
		case "PidFile":
			config.PidFile = val
		case "Mode":
			switch val {
			case "client":
				config.Mode = CLIENT
			case "server":
				config.Mode = SERVER
			case "daemon":
				config.Mode = DAEMON
			case "demon":
				fmt.Println(">:)")
				config.Mode = DAEMON
			default:
				return nil, fmt.Errorf("Config parse error : unknown keyword in line %d for %s. Expected %s or %s", i, key, "client", "server")
			}
		case "Address":
			config.Address = val
		case "MTU":
			mtu, err := strconv.Atoi(val)
			if err != nil {
				return nil, fmt.Errorf("Config parse error : a number expected for mtu ; %s", err)
			}
			config.MTU = mtu
		case "Interface":
			config.InterfaceName = val
		case "DHCP":
			config.DHCP = val
		case "Secret":
			config.Secret = val
		default:
			return nil, fmt.Errorf("Config parse error : garbage values ignored, '%s'", key)
		}

	}
	if config.Address == "" && config.Mode != DAEMON {
		return nil, fmt.Errorf("Config parse error : no address specified")
	}

	if config.PidFile == "" {
		config.PidFile = "/tmp/pin.pid"
	}

	if config.MTU <= 0 {
		config.MTU = 1500
	}

	return config, nil
}
