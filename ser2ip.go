package main

import (
	"fmt"
	"github.com/tarm/goserial"
	"io"
	"net"
	"os"
	"strconv"
)

func main() {
	args := os.Args[1:]
	nrArgs := len(args)

	if nrArgs < 3 {
		fmt.Println("Usage: ser2ip serialportname baudrate port\n")
		return
	}

	portname := args[0]

	baudrate, err := strconv.Atoi(args[1])
	if err != nil || baudrate < 100 {
		fmt.Fprintf(os.Stderr, "Invalid baudrate: ", args[1])
		return
	}

	portnr, err := strconv.Atoi(args[2])
	if err != nil || portnr < 1 || portnr > 65535 {
		fmt.Fprintf(os.Stderr, "Invalid port number: ", args[2])
		return
	}

	fmt.Printf("Starting serial-port to IP converter\n")
	fmt.Printf("Com port: %s, Baudrate: %d, Port: %d\n", portname, baudrate, portnr)

	serConfig := serial.Config{Name: portname, Baud: baudrate}
	serPort, err := serial.OpenPort(&serConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can not open serial port: %s\n", err)
		return
	}
	defer serPort.Close()

	listener, err := net.Listen("tcp", ":"+strconv.Itoa(portnr))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can not open socket server: %s\n", err)
		return
	}
	defer listener.Close()

	ser2ipBuf := make([]byte, 1024)
	ip2serBuf := make([]byte, 1024)

	serPortReadChan := make(chan readResult)
	serPortReadMore := make(chan bool)
	go readProc(serPort, ser2ipBuf, serPortReadChan, serPortReadMore)

	ipReadChan := make(chan readResult)

	acceptChan := make(chan acceptResult)
	acceptMore := make(chan bool)
	go acceptProc(listener, acceptChan, acceptMore)

	// Things that belong to the current connection
	var currentCon net.Conn = nil
	var currentReadMore chan bool = nil
	var connErr error = nil
	var serialErr error = nil

	for {
		select {
		case readResult := <-serPortReadChan:
			if readResult.err != nil {
				serialErr = readResult.err
			} else {
				if currentCon != nil {
					_, connErr = currentCon.Write(ser2ipBuf[0:readResult.bytesRead])
				}
				// Read more
				serPortReadMore <- true
			}
		case readResult := <-ipReadChan:
			if readResult.err != nil {
				// Error reading from ip connection
				connErr = readResult.err
			} else {
				_, serialErr = serPort.Write(ip2serBuf[0:readResult.bytesRead])
				if serialErr == nil {
					// Read more
					currentReadMore <- true
				}
			}
		case acceptResult := <-acceptChan:
			if acceptResult.err != nil {
				fmt.Fprintf(os.Stderr, "Can not accept connection: %s\n", acceptResult.err)
				return
			} else {
				currentCon = acceptResult.conn
				currentReadMore = make(chan bool)
				go readProc(currentCon, ip2serBuf, ipReadChan, currentReadMore)
			}
		}

		if serialErr != nil {
			fmt.Fprintf(os.Stderr, "Error reading from serial port: %s\n", serialErr)
			if currentCon != nil {
				currentCon.Close()
				return
			}
		}

		if currentCon != nil && connErr != nil {
			// Close the connection and accept a new one
			currentCon.Close()
			currentCon = nil
			connErr = nil
			acceptMore <- true
		}
	}
}

type readResult struct {
	bytesRead int
	err       error
}

type acceptResult struct {
	conn net.Conn
	err  error
}

// Reads from a reader and returns the results in a channel
// After that reading will be stopped until readMore is signaled to give the
// receiver a chance to work with everything in the buffer before we overwrite it
func readProc(src io.Reader, buf []byte, result chan readResult, readMore chan bool) {
	for {
		n, err := src.Read(buf)
		result <- readResult{bytesRead: n, err: err}

		_, ok := <-readMore
		if !ok {
			return
		}
	}
}

// Accepts connections in the goroutine
// After accepting a single connection accepting will be stopped until acceptMore is signaled
func acceptProc(listener net.Listener, result chan acceptResult, acceptMore chan bool) {
	for {
		conn, err := listener.Accept()
		result <- acceptResult{conn: conn, err: err}

		_, ok := <-acceptMore
		if !ok {
			return
		}
	}
}
