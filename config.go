package main

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"strings"

	"gitlab.com/aki237/pin/pinlib"
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
	Mode                 RunMode           `yaml:"mode"`
	Address              string            `yaml:"address"`
	MTU                  int               `yaml:"mtu"`
	InterfaceName        string            `yaml:"interfaceName"`
	DHCP                 string            `yaml:"dhcp"`
	DNS                  []string          `yaml:"dns"`
	Secret               string            `yaml:"secret"`
	PostInitScript       map[string]string `yaml:"postServerInit"`
	PostConnectScript    map[string]string `yaml:"postConnect"`
	PostDisconnectScript map[string]string `yaml:"postDisconnect"`
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

// GetSession is used to initialize the pin with right type of session
// (server/client) and peer object.
func (config *Config) GetSession() (*Session, error) {
	server := config.Mode == SERVER
	var err error
	var session *Session = &Session{}
	session.Config = config
	iface := NewTUN(&session.InterfaceName)

	remoteAddress, err := net.ResolveTCPAddr("tcp", session.Address)
	if err != nil {
		return nil, err
	}

	session.RemotePort = remoteAddress.Port
	session.ResolvedRemoteIP = remoteAddress.IP

	secretdec, err := base64.StdEncoding.DecodeString(session.Secret)
	if err != nil {
		return nil, err
	}

	if len(secretdec) != 32 {
		return nil, fmt.Errorf("Error : key length mismatch, need 40 got %d", len(secretdec))
	}

	var kcn [32]byte
	copy(kcn[:], secretdec)

	if !server {
		session.peer = pinlib.NewClient(session.Address, iface, kcn)

		session.SetupClient()
		return session, nil
	}

	var ipNet *net.IPNet
	var ip net.IP
	ip, ipNet, err = net.ParseCIDR(session.DHCP)
	if err != nil {
		return nil, err
	}
	ipNet.IP = ip
	session.peer, err = pinlib.NewServer(session.Address, iface, ipNet, kcn)
	if err != nil {
		return nil, err
	}

	return session, session.SetupServer()
}
