package sheriff

import (
	"bufio"
	"encoding/json"
	"fmt"
	"mymodule/config"
	elev "mymodule/elevator"
	"mymodule/network/peers"
	"net"
	"strconv"
	"strings"
)

type Message struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

var Connections = make(map[string]net.Conn)
var DeputyIDChan = make(chan string)
var DeputyUpdateChan = make(chan bool)

func Sheriff(incomingOrder chan elev.Orderstatus, networkOrders map[string]elev.Orderstatus, nodeLeftNetwork chan string) {
	ipID := strings.Split(string(config.Self_id), ":")
	go peers.Transmitter(config.Sheriff_port, ipID[0], make(chan bool)) //channel for turning off sheriff transmitt?
	//go peers.Receiver(15647, peerUpdateCh)
	go deputyUpdater(networkOrders)
	go listenForConnections(incomingOrder, nodeLeftNetwork)
}

func deputyUpdater(networkOrders map[string]elev.Orderstatus) {
	var deputyID string
	for {
		select {
		case <-DeputyUpdateChan:
			//check if deputy is in the list of connections
			if _, ok := Connections[deputyID]; !ok {
				deputyID = ChooseNewDeputy()
			}
			//send all orders to deputy
			SendDeputyMessage(deputyID, networkOrders)
		}
	}
}

func listenForConnections(incomingOrder chan elev.Orderstatus, nodeLeftNetwork chan string) {
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

		var peerID string
		if _, ok := Connections[strings.Split(conn.RemoteAddr().String(), ":")[0]+":1"]; ok {
			peerID = strings.Split(conn.RemoteAddr().String(), ":")[0] + ":2"
			Connections[peerID] = conn
		} else {
			peerID = strings.Split(conn.RemoteAddr().String(), ":")[0] + ":1"
			Connections[peerID] = conn
		}

		fmt.Println("Accepted connection from", conn.RemoteAddr())
		//DeputyUpdateChan <- true
		go ReceiveMessage(conn, incomingOrder, peerID, nodeLeftNetwork)

	}
}

// func sheriffUpdater(peerUpdateCh chan peers.PeerUpdate, world *assigner.World) {

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

func SendOrderMessage(peer string, order elev.Orderstatus) (bool, error) {
	//ip := strings.Split(peer, ":")[0]
	fmt.Println("Connections:", Connections)
	fmt.Println("Peer:", peer)
	conn, ok := Connections[peer]

	if !ok {
		fmt.Println("No connection to peer", peer)
		return false, fmt.Errorf("no connection to peer %s", peer)
	}

	// Convert the order to JSON
	orderJSON, err := json.Marshal(order)
	if err != nil {
		fmt.Println("Error marshalling order:", err)
		return false, err
	}

	// Create a new message with type "order"
	msg := Message{
		Type: "order",
		Data: orderJSON,
	}

	// Convert the message to JSON
	msgJSON, err := json.Marshal(msg)
	if err != nil {
		// ...
	}

	_, err = fmt.Fprintln(conn, string(msgJSON))
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

func SendDeputyMessage(peer string, nodeOrders map[string]elev.Orderstatus) (bool, error) {

	conn, ok := Connections[peer]

	if !ok {
		fmt.Println("No connection to peer", peer)
		return false, fmt.Errorf("no connection to peer %s", peer)
	}

	//convert the map to JSON
	nodeOrdersJSON, err := json.Marshal(nodeOrders)
	if err != nil {
		fmt.Println("Error marshalling node orders to be sent to deputy:", err)
		return false, err
	}

	// Create a new message with type "deputy"
	msg := Message{
		Type: "deputyMessage",
		Data: nodeOrdersJSON,
	}

	// Convert the message to JSON
	msgJSON, err := json.Marshal(msg)
	if err != nil {
		fmt.Println("Error marshalling deputy message:", err)
	}

	_, err = fmt.Fprintln(conn, string(msgJSON))
	if err != nil {
		fmt.Println("Error sending node orders to deputy, he might be dead:", err)

		return false, err
	}

	fmt.Println("Sent node orders to deputy.", peer)
	return true, nil

}

func ReceiveMessage(conn net.Conn, incomingOrder chan elev.Orderstatus, peerID string, nodeLeftNetwork chan string) (elev.Orderstatus, error) {
	for {
		reader := bufio.NewReader(conn)
		message, err := reader.ReadString('\n')
		if err != nil {

			if err.Error() == "EOF" {
				fmt.Println("Connection closed by", peerID)

				conn.Close()
				//DeputyUpdateChan <- true
				nodeLeftNetwork <- peerID
				delete(Connections, peerID)
				return elev.Orderstatus{}, nil

			} else {
				fmt.Println("Error reading from connection:", err)
			}
			continue
		}

		// Convert the message from JSON to elev.Orderstatus
		var order elev.Orderstatus
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

		fmt.Println("Received order from", peerID)

		//DeputyUpdateChan <- true

		incomingOrder <- order

	}

}

func ChooseNewDeputy() string {
	//fmt.Println("Choosing new deputy")
	for k := range Connections {
		fmt.Println("The new deputy is:", k)
		return k
	}
	return ""
}
