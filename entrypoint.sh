#!/bin/sh
# Start chisel server for TCP tunneling
chisel server --port 20081 --reverse &
# Start remote-open server for URL forwarding
exec remote-open-server --port 20080
