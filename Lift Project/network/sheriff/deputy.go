package sheriff

import (
	"bufio"
	"encoding/json"
	"fmt"
	"mymodule/config"
	. "mymodule/types"
	"net"
	"time"
)

var receivedDeputyNodeOrders = make(chan map[string]Orderstatus)
var sheriffDisconnected = make(chan net.Conn)
var DeputyBecomeSheriff = make(chan map[string]Orderstatus)

func initDeputy() {
	sheriffIP := GetSheriffIP()
	dep2SherConn, err := connectDeputyToSheriff(sheriffIP)
	if err != nil {
		fmt.Println("Error connecting to sheriff:", err)
		return
	}
	go sheriffHandler()
	go receiveDeputyMessage(dep2SherConn)
}

func connectDeputyToSheriff(sheriffIP string) (net.Conn, error) {
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", sheriffIP, config.Sheriff_deputy_port))
	if err != nil {
		fmt.Println("Error connecting to sheriff:", err)
		return nil, err
	}
	fmt.Println("Me, a deputy, connected to sheriff")

	return conn, nil
}

func sheriffHandler() {
	var deputyNodeOrders map[string]Orderstatus

	for {
		select {
		case orders := <-receivedDeputyNodeOrders:
			deputyNodeOrders = orders

		case <-sheriffDisconnected:
			fmt.Println("Sheriff disconnected")
			becomeSheriff(deputyNodeOrders) // How to become sheriff?

		}
	}
}

func receiveDeputyMessage(deputyToSheriffConn net.Conn) {
	readErrors := 0
	for readErrors < 3 {
		reader := bufio.NewReader(deputyToSheriffConn)
		message, err := reader.ReadString('\n')
		//fmt.Println("Received message from sheriff:", message)
		if err != nil {
			fmt.Println("Error reading from sheriff:", err)
			time.Sleep(2 * time.Second)
			readErrors++
			continue
		}

		var msg Message
		err = json.Unmarshal([]byte(message), &msg)
		if err != nil {
			fmt.Println("Error parsing message:", err)

			continue
		}

		switch msg.Type {
		case "deputyMessage":
			var deputyNodeOrders map[string]Orderstatus
			err = json.Unmarshal(msg.Data, &deputyNodeOrders)
			if err != nil {
				fmt.Println("Error parsing deputy message:", err)
				continue
			}
			// Handle deputy message...
			fmt.Println("Received deputy message from sheriff")

		default:
			fmt.Println("Unknown message type:", msg.Type)
		}
	}
	sheriffDisconnected <- deputyToSheriffConn
}

func becomeSheriff(deputyNodeOrders map[string]Orderstatus) {
	//Dont know how right now
	fmt.Println("Theres a new sheriff in town, I killed the old one")
	fmt.Println("but I dont know how to become the sheriff yet.....")
	DeputyBecomeSheriff <- deputyNodeOrders // Not sure if this is the right way to do it
}
