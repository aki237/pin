package main

import (
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
)

// RunMode specifies the mode in which the program is run as
type RunMode int

// All the available RunModes
const (
	SERVER RunMode = iota
	CLIENT
)

// Config struct is used to store the values parsed from the config file
type Config struct {
	Mode          RunMode
	Address       string
	MTU           int
	InterfaceName string
	DHCP          string
	Secret        string
}

// NewConfigFromFile is used to read configuration data from provided filename and return a Config struct
// after parsing the contents
func NewConfigFromFile(filename string) (*Config, error) {
	contents, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(contents), "\n")

	config := &Config{}

	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		vals := strings.SplitN(line, ":", 2)
		if len(vals) != 2 {
			return nil, fmt.Errorf("Config parse error : error in line %d. Expected a key palue pair separated by ':'", i)
		}
		key := strings.TrimSpace(vals[0])
		val := strings.TrimSpace(vals[1])
		err = config.setValueForKey(i, key, val)
		if err != nil {
			return nil, fmt.Errorf("Config parse error : error in line %d. Expected a key palue pair separated by ':'", i)
		}
	}
	if config.Address == "" {
		return nil, fmt.Errorf("Config parse error : no address specified")
	}

	if config.MTU <= 0 {
		config.MTU = 1500
	}

	return config, nil
}

func (config *Config) setValueForKey(i int, key, val string) error {
	switch key {
	case "Mode":
		switch val {
		case "client":
			config.Mode = CLIENT
		case "server":
			config.Mode = SERVER
		default:
			return fmt.Errorf("Config parse error : unknown keyword in line %d for %s. Expected %s or %s", i, key, "client", "server")
		}
	case "Address":
		config.Address = val
	case "MTU":
		mtu, err := strconv.Atoi(val)
		if err != nil {
			return fmt.Errorf("Config parse error : a number expected for mtu ; %s", err)
		}
		config.MTU = mtu
	case "Interface":
		config.InterfaceName = val
	case "DHCP":
		config.DHCP = val
	case "Secret":
		config.Secret = val
	default:
		return fmt.Errorf("Config parse error : garbage values ignored, '%s'", key)
	}
	return nil
}
