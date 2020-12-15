# pin

[![Build Status](https://gitlab.com/aki237/pin/badges/master/build.svg)](https://gitlab.com/aki237/pin/-/jobs)

pin a simple tunnel client and server which is configured to act as a VPN by default.
It is tested and known to work in Linux (obviously), FreeBSD , DragonflyBSD, NetBSD and OpenBSD*

* *OpenBSD : might have to check again*

## Usage

In both the ends (server and the client), the usage is simple :

```
# pin -c /path/to/config/file
```

## Configuration Syntax

The configuration syntax is just yaml:

```yaml
# Comments are awesome...
# Commented line begins with a hash
#
# What mode to run as : client server
mode : server
#
# For clients, Address is the info of the remote server. Example : [PROTO]://12.13.14.15:9090
# For servers, it is the address to listen at. Example : [PROTO]://0.0.0.0:9090 (you know listen at all interfaces stuff...)
# The PROTO argument can be either `tcp` or `udp`
address : udp://pin.erred.dev:9090
#
# For the serious folks, you can set the MTU for optimised speed or CPU usage
# If this is to be changed, pls change it for the tunneling interface too.
# If you have no idea what this does, just leave it as 1500 or just don't specify this.
mtu : 1500
#
# What should be the name of the interface. This only works in Linux.
# Seriously you can go wild... I tried poniesareawesome... But apparently
# IFNAMSIZ in /usr/include/linux/if.h said only 16 characters allowed :'(
# So I got only "poniesareawesom". (\0 makes it 16.)
# For other platforms, you are stuck with tun{\d}. Sad??
interface : pin0
#
# Ok Let's get serious. This is a secret key. Shared by both server and 
# the client (Symmetric). How to generate this secret??? That will be stated down below.
secret : u7ZQZWomGPHG0GKqoe8E7Vg+hgIxiYnn7Yr4HBz4VWs=
#
# For the server folks... You know what this is. DHCP.. No not an actual DHCP running inside.
# But for provision and Connection Identification.
dhcp : 10.10.0.1/24
#
# For the client side, optionally DNS can be setup by using the DNS option
# Multiple DNS server IPs can be specified by separating them with a comma like the following :
dns : 
  - 1.1.1.1
  - 8.8.8.8
  - 4.4.2.2

# All the post init stuff is moved out of the Go code.
# Now `pin` exports all the required variables which will be helpful to
# write iptables (pf/ip/[any tool]) rules. The following sections will contain
# platform specific scripting. For example in postServerInit if linux, freebsd
# and openbsd are defined, then only the script defined for the platform which pin
# is running on will be executed.

# postServerInit should contains the shell script template (see Go templates)
# which will be run after the pin VPN server is started. See setup.go#L77 to see
# the list of all the exported variables.
postServerInit:
  linux: |
    sysctl -w net.ipv4.ip_forward=1
    ip link set dev {{.interfaceName}}  mtu {{.mtu}}
    ip link set {{.interfaceName}} up

    ip addr add {{.tunIP}} dev {{.interfaceName}}

    export DEFAULT_LINK=$(ip route | awk '/default/ { print $5 }')
    iptables -F
    iptables -F -t nat
    iptables -I FORWARD -i {{.interfaceName}} -j ACCEPT
    iptables -I FORWARD -o {{.interfaceName}} -j ACCEPT
    iptables -I INPUT -i {{.interfaceName}} -j ACCEPT
    iptables -t nat -I POSTROUTING -o $DEFAULT_LINK -j MASQUERADE

# Similar to postServerInit postConnect is the shell script that runs in the client
# system after the local tun device is initialized and connection to the remote is
# established
postConnect:
  linux: |
    ip link set dev {{.interfaceName}}  mtu {{.mtu}}
    ip link set {{.interfaceName}} up

    export DEFAULT_GW=$(ip route | awk '/default/ { print $3 }')
    ip route add {{.remoteIP}} via $DEFAULT_GW

    ip addr add {{.tunIP}} dev {{.interfaceName}}
    ip route add 0.0.0.0/1 via {{.tunGateway}}
    ip route add 128.0.0.0/1 via {{.tunGateway}}
    cp /etc/resolv.conf /tmp/pin.resolv.conf.bckup
    echo > /etc/resolv.conf
    {{range $dnsip := .dns}}
      echo "nameserver {{$dnsip}}" >> /etc/resolv.conf
    {{end}}

# postDisconnect is run after the client is disconnected from the VPN due to any reason
# Be it Ctrl-C or remote is not reachable anymore.
postDisconnect:
  linux: |
    ip route del {{.remoteIP}}
    cp /tmp/pin.resolv.conf.bckup /etc/resolv.conf
```

# Secret Generation

The program uses, ChaCha20Poly1305 as the encryption algorithm. Which requires a 32 byte key.

So let's generate a key.
(If you didn't notice, that key specified is a base64 encoded string.)

```shell
$ dd status=none if=/dev/urandom of=/dev/stdout bs=1 count=32 | base64
u7ZQZWomGPHG0GKqoe8E7Vg+hgIxiYnn7Yr4HBz4VWs=
```

This key is to be shared by both the server and the client.

# Disclaimer

This is a hobby project. I'm neither a security expert or a network expert.
 * Google and SO said me salsa20 is good enough for daily use and easy on CPU.
 * Had to add a MAC for the data sent. So used ChaCha20 (salsa20 family) + Poly1305 combination.
 * Found snappy to be really fast in my experience 
   + Comparing with lzo, zlib, zstd, gzip etc.,
   + But... Man!! that chokes CPU for heavy loads...

This works for me in my university. Feel free to fork it, modify it, use it and contribute too...

# Roadmap
 + ~~Unique nonce generation for every client (connection)~~
 + ~~Add a message authentication layer for integrity~~
 + Clean the code
   - ~~Move post init stuff out of code into shell scripts~~
   - Organize the Session struct and better os signal handling
   - cleanup pinlib for client code
 + User based secret authentication
 + Time based key variation.

# Contributors
 + [aki237](https://gitlab.com/aki237)
 + [sbioa1234](https://gitlab.com/sbioa1234)

