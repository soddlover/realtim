package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
)

func main() {
	// Define the server address
	serverAddr := "10.100.23.129:33546"

	// Connect to the server
	conn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		fmt.Println("Error connecting to the server:", err)
		os.Exit(1)
	}
	defer conn.Close()

	// Send a message to the server
	message := "Connect to: 10.100.23.34:33546!\000"
	_, err = conn.Write([]byte(message))
	if err != nil {
		fmt.Println("Error sending message to the server:", err)
		os.Exit(1)
	}

	// Receive the echoed message back from the server
	reader := bufio.NewReader(conn)
	response, err := reader.ReadString('\000')
	if err != nil {
		fmt.Println("Error reading response from the server:", err)
		os.Exit(1)
	}

	// Print the echoed message
	fmt.Println("Received from server:", response)

	// Send another message to the server
	message = "Another message!\000"
	_, err = conn.Write([]byte(message))
	if err != nil {
		fmt.Println("Error sending message to the server:", err)
		os.Exit(1)
	}

	// Receive the echoed message back from the server
	response, err = reader.ReadString('\000')
	if err != nil {
		fmt.Println("Error reading response from the server:", err)
		os.Exit(1)
	}

	// Print the echoed message
	fmt.Println("Received from server:", response)
}
