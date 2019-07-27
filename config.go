package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"strings"

	"gopkg.in/yaml.v2"
)

// RunMode specifies the mode in which the program is run as
type RunMode int

// String method implements the stringer interface
func (r RunMode) String() string {
	if r == SERVER {
		return "server"
	}

	return "client"
}

// UnmarshalYAML implements the yaml.Unmarshaler interface for this enum type.
func (r *RunMode) UnmarshalYAML(unmarshal func(interface{}) error) error {
	x := ""
	if err := unmarshal(&x); err != nil {
		return err
	}

	fmt.Println("Debug::YAMLUnmarshal: runMode : ", x)
	switch strings.ToLower(x) {
	case "server":
		*r = SERVER
	case "client":
		*r = CLIENT
	default:
		return errors.New("invalid runMode passed: expects either 'client' or 'server'")
	}

	return nil
}

// All the available RunModes
const (
	SERVER RunMode = iota
	CLIENT
)

// Config struct is used to store the values parsed from the config file
type Config struct {
	Mode          RunMode  `yaml:"mode"`
	Address       string   `yaml:"address"`
	MTU           int      `yaml:"mtu"`
	InterfaceName string   `yaml:"interfaceName"`
	DHCP          string   `yaml:"dhcp"`
	DNS           []string `yaml:"dns"`
	Secret        string   `yaml:"secret"`
}

// NewConfigFromFile is used to read configuration data from provided filename and return a Config struct
// after parsing the contents
func NewConfigFromFile(filename string) (*Config, error) {
	contents, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	config := &Config{}

	err = yaml.Unmarshal(contents, config)
	if err != nil {
		return nil, err
	}

	if config.Address == "" {
		return nil, fmt.Errorf("Config parse error : no address specified")
	}

	if config.MTU <= 0 {
		config.MTU = 1500
	}

	return config, nil
}
