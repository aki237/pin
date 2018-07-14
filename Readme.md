# pin

![Build Status](https://gitlab.com/aki237/pin/badges/master/build.svg)

pin a simple tunnel client and server which is configured to act as a VPN by default.
It is tested and known to work in Linux (obviously), FreeBSD , DragonflyBSD, NetBSD and OpenBSD*

It used to work on windows with the TUN/TAP driver. But again the routing system and the inconsistencies of windows
seriously pissed me off, so killed its support.

* *OpenBSD : might have to check again*

## Usage

In both the ends (server and the client), the usage is simple :

```
# pin -c /path/to/config/file
```

## Configuration Syntax

The configuration syntax is simple (as usual :P):

```conf
# Comments are awesome...
# Commented line begins with a hash
#
# What mode to run as : client server
Mode : client
#
# For clients, Address is the info of the remote server. Example : 12.13.14.15:9090
# For servers, it is the address to listen at. Example : 0.0.0.0:9090 (you know listen at all interfaces stuff...)
Address : addre.ss:9067
#
# For the serious folks, you can set the MTU for optimised speed or CPU usage
# If this is to be changed, pls change it for the tunneling interface too.
# If you have no idea what this does, just leave it as 1500 or just don't specify this.
MTU : 1500
#
# What should be the name of the interface. This only works in Linux.
# Seriously you can go wild... I tried poniesareawesome... But apparently
# IFNAMSIZ in /usr/include/linux/if.h said only 16 characters allowed :'(
# So I got only "poniesareawesom". (\0 makes it 16.)
# For other platforms, you are stuck with tun{\d}. Sad??
Interface : pin0
#
# Ok Let's get serious. This is a secret key. Shared by both server and 
# the client (Symmetric). How to generate this secret??? That will be stated down below.
Secret : u7ZQZWomGPHG0GKqoe8E7Vg+hgIxiYnn7Yr4HBz4VWs=
#
# For the server folks... You know what this is. DHCP.. No not an actual DHCP running inside.
# But for provision and Connection Identification.
DHCP : 10.10.0.1/24
```

# Secret Generation

The program uses, ChaCha20 as the encryption algorithm. Which requires a 32 byte key.

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
 + Time based key variation.

# Contributors
 + [aki237](https://gitlab.com/aki237)
 + [sbioa1234](https://gitlab.com/sbioa1234)

