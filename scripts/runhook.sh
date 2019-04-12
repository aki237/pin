#!/bin/sh

FILENAME="/server.pin"

# Create the config file
echo > $FILENAME
echo "Mode : server" >> $FILENAME
echo "Address : 0.0.0.0:9090" >> $FILENAME
echo "MTU : 1500" >> $FILENAME
echo "Interface : pin0" >> $FILENAME
echo "DHCP : 10.0.0.1/24" >> $FILENAME
if [ "$SECRET" == "" ]; then
    echo "Secret not passed in the environment variable"
    exit 20
fi
echo "Secret : $SECRET" >> $FILENAME

# Initialize the kernel tun device
mkdir -p /dev/net
mknod /dev/net/tun c 10 200

# Unleash the f*cking dragon
/pin $FILENAME
