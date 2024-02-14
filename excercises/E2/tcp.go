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
	err = conn.SetNoDelay(true)

	reader := bufio.NewReader(conn)
	data, err := reader.ReadBytes('\000')
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Print("pooop> ", string(data))

	_, err = conn.Write([]byte("wanker\000"))
	fmt.Println("send connect...")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	response, err := reader.ReadBytes('\000')
	fmt.Println(string(response))

}
