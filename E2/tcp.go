package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
)

func main() {

	if len(os.Args) == 1 {
		fmt.Println("please")
		os.Exit(1)
	}

	tcpAddr, err := net.ResolveTCPAddr("tcp4", os.Args[1])

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	conn, err := net.DialTCP("tcp", nil, tcpAddr)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	_, err = conn.Write([]byte("Connect to: 10.100.23.34:33546\000"))
	fmt.Println("send connect...")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	reader := bufio.NewReader(conn)
	data, err := reader.ReadString('\000')
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Print("> ", string(data))

	_, err = conn.Write([]byte("halla\000"))
	fmt.Println("send message...")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	data, err = reader.ReadString('\000')

	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Print("> ", string(data))

}
