package sheriff

import (
	"bufio"
	"encoding/json"
	"fmt"
	"mymodule/config"
	"mymodule/network/peers"
	. "mymodule/types"
	"net"
	"strconv"
	"strings"
	"time"
)

const DEPUTY_SEND_FREQ = 3 * time.Second

var WranglerConnections = make(map[string]net.Conn)

var NewDeputyConnChan = make(chan net.TCPConn)
var DeputyDisconnectChan = make(chan net.TCPConn)

func CheckMissingConnToOrders(networkOrders map[string]Orderstatus, nodeLeftNetwork chan string) {
	processedIDs := make(map[string]bool)

	for _, order := range networkOrders {
		id := order.Owner
		if WranglerConnections[id] == nil && id != config.Self_id && !processedIDs[id] {
			nodeLeftNetwork <- id
			fmt.Println("***Missing connection to ACTIVE ORDER Reassigning order!!!***", id)
			processedIDs[id] = true
		}
	}
}
func Sheriff(incomingOrder chan Orderstatus, networkOrders map[string]Orderstatus, nodeLeftNetwork chan string) {
	ipID := strings.Split(string(config.Self_id), ":")
	go peers.Transmitter(config.Sheriff_port, ipID[0], make(chan bool)) //channel for turning off sheriff transmitt?
	//go peers.Receiver(15647, peerUpdateCh)
	go listenForWranglerConnections(incomingOrder, nodeLeftNetwork)
	go SendNodeOrdersToDeputy(networkOrders)
	time.Sleep(1 * time.Second)
	CheckMissingConnToOrders(networkOrders, nodeLeftNetwork)
}

func listenForWranglerConnections(incomingOrder chan Orderstatus, nodeLeftNetwork chan string) {
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
		reader := bufio.NewReader(conn)
		message, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading from connection:", err)
			continue
		}

		peerID := strings.TrimSpace(message)

		WranglerConnections[peerID] = conn

		fmt.Println("Accepted Wrangler", peerID)
		fmt.Println(WranglerConnections)
		go ReceiveMessage(conn, incomingOrder, peerID, nodeLeftNetwork)

	}
}

func SendNodeOrdersToDeputy(nodeOrders map[string]Orderstatus) {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			SendDeputyMessage(nodeOrders)
			//add updatechan
		}
	}
}

func SendDeputyMessage(nodeOrders map[string]Orderstatus) {
	var chosenOneID string
	for id, conn := range WranglerConnections {
		if chosenOneID == "" || WranglerConnections[chosenOneID] == nil {
			chosenOneID = id
		}
		nodeOrdersData := NodeOrdersData{
			NodeOrders:   nodeOrders,
			TheChosenOne: id == chosenOneID, // or false, depending on your logic
		}
		nodeOrdersDataJSON, err := json.Marshal(nodeOrdersData)
		if err != nil {
			fmt.Println("Error marshalling node orders to be sent to deputy:", err)
		}

		// Create a new message with type "deputy"
		msg := Message{
			Type: "NodeOrders",
			Data: nodeOrdersDataJSON,
		}

		// Convert the message to JSON
		msgJSON, err := json.Marshal(msg)
		if err != nil {
			fmt.Println("Error marshalling deputy message:", err)
		}

		_, err = fmt.Fprintln(conn, string(msgJSON))
		if err != nil {
			fmt.Println("Error sending node orders to deputy, he might be dead:", err)
			//deputyConn.Close()
			//DeputyDisconnectChan <- deputyConn
		}

		fmt.Println("Sent node orders to deputy.")

	}
}

func SendOrderMessage(peer string, order Orderstatus) (bool, error) {
	//ip := strings.Split(peer, ":")[0]
	fmt.Println("Connections:", WranglerConnections)
	fmt.Println("Peer:", peer)
	conn, ok := WranglerConnections[peer]

	if !ok {
		fmt.Println("No connection to ", peer)

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
		conn.Close()
		delete(WranglerConnections, peer)
		return false, err
	}
	fmt.Println("successs Sent order to", peer, "order:", order)
	return true, nil
}

func ReceiveMessage(conn net.Conn, incomingOrder chan Orderstatus, peerID string, nodeLeftNetwork chan string) (Orderstatus, error) {
	for {
		reader := bufio.NewReader(conn)
		message, err := reader.ReadString('\n')
		if err != nil {

			if err.Error() == "EOF" {
				fmt.Println("Connection closed by", peerID)

				conn.Close()
				nodeLeftNetwork <- peerID
				delete(WranglerConnections, peerID)
				return Orderstatus{}, nil

			} else {
				fmt.Println("Error reading from connection:", err)
			}
			continue
		}

		// Convert the message from JSON to Orderstatus
		var order Orderstatus
		err = json.Unmarshal([]byte(message), &order)
		if err != nil {
			fmt.Println("Error unmarshalling order:", err)
			continue
		}

		fmt.Println("Received order from", peerID)

		//DeputyUpdateChan <- true

		incomingOrder <- order

	}
}

func ChooseNewDeputy() (net.Conn, string, error) {
	//fmt.Println("Choosing new deputy")
	for k := range WranglerConnections {
		fmt.Println("The new deputy is:", k)
		return WranglerConnections[k], k, nil
	}
	return nil, "", fmt.Errorf("no wrangler connections")
}
