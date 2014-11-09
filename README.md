ser2ip
======

A simple full-duplex serial-port to TCP/IP gateway in Go

This daemon allows to expose a serial port through a socket over the network.

The server socket will only accept a single TCP/IP connection at a time because
a serialport is considered as an exclusive resoure. When the client quits a new
one can be accepted.

If the serialport is not available or encounters an error this application will
stop.

The serialport which should be exposed and the port on which it should be
exposed can be configured through commandline parameters.

###Usage:

~~~~bash
ser2ip serialPortName baudrate ipPortNumber
~~~~

###Examples:

~~~~bash
# Exposing COM3 with baudrate 115200 on port 9000
ser2ip COM3 115200 9000

# Exposing /dev/ttyUSB0 with baudrate 9600 on port 9001
ser2ip /dev/ttyUSB0 9600 9001
~~~~
