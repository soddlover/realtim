package supervisor

import (
	"bufio"
	"encoding/json"
	"fmt"
	"mymodule/config"
	"mymodule/elevator/elevio"
	"mymodule/network/peers"
	"net"
	"strconv"
	"strings"
)

type Orderstatus struct {
	Owner   string
	OrderID string
	Floor   int
	Button  elevio.ButtonType
	Status  bool
}

var NodeOrders = make(map[string]Orderstatus)
var Connections = make(map[string]net.Conn)

func Supervisor(incomingOrder chan Orderstatus) {
	ipID := strings.Split(string(config.Self_id), ":")
	go peers.Transmitter(config.Supervisor_port, ipID[0], make(chan bool)) //channel for turning off supervisor transmitt?
	//go peers.Receiver(15647, peerUpdateCh)
	go listenForConnections(incomingOrder)
}

func listenForConnections(incomingOrder chan Orderstatus) {
	ln, err := net.Listen("tcp", ":"+strconv.Itoa(config.TCP_port))
	if err != nil {
		fmt.Println("Error listening for connections:", err)
		return
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}
		Connections[strings.Split(conn.RemoteAddr().String(), ":")[0]] = conn
		fmt.Println("Accepted connection from", conn.RemoteAddr())
		go ReceiveMessage(conn, incomingOrder)

	}
}

// func supervisorUpdater(peerUpdateCh chan peers.PeerUpdate, world *assigner.World) {

// 	for {
// 		select {
// 		case p := <-peerUpdateCh:
// 			// Connect to new peers
// 			for _, newPeer := range p.New {
// 				ipID := strings.Split(string(newPeer), ":")
// 				IDint, err := strconv.Atoi(ipID[1])
// 				if err != nil {
// 					fmt.Println("Error converting string to int:", err)
// 					return
// 				}
// 				port := config.TCP_port + IDint
// 				conn, err := net.Dial("tcp", string(ipID[0])+":"+string(port))
// 				if err != nil {
// 					fmt.Println("Error connecting to peer:", err)
// 					continue
// 				}
// 				Connections[string(newPeer)] = conn
// 			}

// 			// Close connections to lost peers
// 			for _, lostPeer := range p.Lost {
// 				if conn, ok := Connections[lostPeer]; ok {
// 					conn.Close()
// 					delete(Connections, lostPeer)
// 				}
// 			}
// 		}
// 	}
// }

func SendMessage(peer string, order Orderstatus) (bool, error) {
	ip := strings.Split(peer, ":")[0]
	conn, ok := Connections[ip]
	if !ok {
		return false, fmt.Errorf("no connection to peer %s", peer)
	}

	// Convert the order to JSON
	orderJSON, err := json.Marshal(order)
	if err != nil {
		fmt.Println("Error marshalling order:", err)
		return false, err
	}

	_, err = fmt.Fprintln(conn, string(orderJSON))
	if err != nil {
		fmt.Println("Error sending order:", err)
		return false, err
	}

	// // Wait for an acknowledgement
	// reader := bufio.NewReader(conn)
	// acknowledgement, err := reader.ReadString('\n')
	// if err != nil {
	// 	return false, err
	// }

	// // Check the acknowledgement
	// if strings.TrimSpace(acknowledgement) != "ACK" {
	// 	return false, fmt.Errorf("unexpected acknowledgement: %s", acknowledgement)
	// }
	fmt.Println("successs Sent order to", peer, "order:", order)
	return true, nil
}

func ReceiveMessage(conn net.Conn, incomingOrder chan Orderstatus) (Orderstatus, error) {
	for {
		reader := bufio.NewReader(conn)
		message, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading from connection:", err)
			continue
		}

		// Convert the message from JSON to Orderstatus
		var order Orderstatus
		err = json.Unmarshal([]byte(message), &order)
		if err != nil {
			fmt.Println("Error unmarshalling order:", err)
			continue
		}

		// // Send an acknowledgement
		// _, err = fmt.Fprintln(conn, "ACK")
		// if err != nil {
		// 	fmt.Println("Error sending acknowledgement:", err)
		// 	continue
		// }
		NodeOrders[order.Owner] = order
		fmt.Println("Received order from", conn.RemoteAddr(), "order:", order)
		incomingOrder <- order
	}

}
