package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
)

func main() {

	if len(os.Args) == 1 {
		fmt.Println("Please provoide bsedbdsfbfb")
		os.Exit(1)
	}

	udpAddr, err := net.ResolveUDPAddr("udp", os.Args[1])

	conn, err := net.DialUDP("udp", nil, udpAddr)

	data, err := bufio.NewReader(conn).ReadString('\n')

	fmt.Print("> ", string(data))
	fmt.Print(err)

}
