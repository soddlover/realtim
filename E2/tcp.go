package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
)

func main() {

	if len(os.Args)==1 {
		fmt.Println("please")
		os.Exit(1)
	}

	tcpAddr, err := net.ResolveTCPAddr
}
