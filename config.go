package main

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/url"
	"strconv"
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

// Protocol is a enum type defining all the supported protocol types
type Protocol string

// All the supported protocol types
const (
	TCP Protocol = "tcp"
	UDP          = "udp"
)

// Address contains the parsed address of the passed
type Address struct {
	Protocol Protocol
	Host     string
	IP       net.IP
	Port     int
}

// UnmarshalYAML implements the yaml.Unmarshaler interface for the Address struct
func (a *Address) UnmarshalYAML(unmarshal func(interface{}) error) error {
	x := ""
	if err := unmarshal(&x); err != nil {
		return err
	}

	uri, err := url.Parse(x)
	if err != nil {
		return errors.New("invalid address passed")
	}

	a.Host = uri.Hostname()
	if a.Host == "" {
		return fmt.Errorf("invalid server address: host not found")
	}

	ipaddr, err := net.ResolveIPAddr("ip", a.Host)
	if err != nil {
		return err
	}
	a.IP = ipaddr.IP

	a.Port, _ = strconv.Atoi(uri.Port())
	if a.Port == 0 {
		return fmt.Errorf("invalid server address: port not found")
	}

	switch uri.Scheme {
	case "tcp", "TCP":
		a.Protocol = TCP
	case "udp", "UDP":
		a.Protocol = UDP
	default:
		return fmt.Errorf("invalid protocol passed: expects either 'tcp' or 'udp', got %s", uri.Scheme)
	}

	return nil
}

// Config struct is used to store the values parsed from the config file
type Config struct {
	Mode                 RunMode           `yaml:"mode"`
	Address              Address           `yaml:"address"`
	MTU                  int               `yaml:"mtu"`
	InterfaceName        string            `yaml:"interfaceName"`
	DHCP                 string            `yaml:"dhcp"`
	DNS                  []string          `yaml:"dns"`
	Secret               string            `yaml:"secret"`
	Motd                 string            `yaml:"motd"`
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

	session.RemotePort = config.Address.Port
	session.ResolvedRemoteIP = config.Address.IP

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
		session.peer = pinlib.NewClient(session.Address.IP, session.Address.Port, string(session.Address.Protocol), iface, kcn)

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
	peer, err := pinlib.NewServer(session.Address.IP, session.Address.Port, string(session.Address.Protocol), iface, ipNet, kcn)
	if err != nil {
		return nil, err
	}

	motd := [128]byte{}
	copy(motd[:], config.Motd)

	peer.SetMotd(motd)

	session.peer = peer

	return session, session.SetupServer()
}
