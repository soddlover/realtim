package sheriff

import (
	"bufio"
	"encoding/json"
	"errors"
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

func CheckMissingConnToOrders(networkOrders *NetworkOrders, nodeLeftNetwork chan<- string) {
	processedIDs := make(map[string]bool)
	fmt.Println("Checking for missing connections to orders")
	networkOrders.Mutex.Lock()
	for floor := 0; floor < len(networkOrders.Orders); floor++ {
		for button := 0; button < len(networkOrders.Orders[floor]); button++ {
			id := networkOrders.Orders[floor][button]
			//fmt.Printf("Checking order at floor %d, button %d, id: %s\n", floor, button, id) // Print the current order being checked
			if id != "" && WranglerConnections[id] == nil && id != config.Self_id && !processedIDs[id] {
				networkOrders.Mutex.Unlock()
				nodeLeftNetwork <- id
				networkOrders.Mutex.Lock()
				fmt.Println("***Missing connection to ACTIVE ORDER Reassigning order!!!***", id)
				processedIDs[id] = true
			} else {
				//fmt.Printf("Order at floor %d, button %d is not missing connection\n", floor, button) // Print a message if the order is not missing connection
			}
		}
	}
	networkOrders.Mutex.Unlock()
}

func Sheriff(
	incomingOrder chan<- Orderstatus,
	networkOrders *NetworkOrders,
	nodeLeftNetwork chan string,
	nodeOrdersUpdateChan chan bool,
	relievedOfDuty <-chan bool,
	quitAssigner chan<- bool) {

	ipID := strings.Split(string(config.Self_id), ":")
	transmitEnable := make(chan bool)
	listenWranglerEnable := make(chan bool)
	sendOrderToDeputyEnable := make(chan bool)
	go peers.Transmitter(config.Sheriff_port, ipID[0], transmitEnable) //channel for turning off sheriff transmitt?
	//go peers.Receiver(15647, peerUpdateCh)
	go listenForWranglerConnections(incomingOrder, nodeLeftNetwork, listenWranglerEnable)
	go SendNodeOrdersToDeputy(networkOrders, nodeOrdersUpdateChan, sendOrderToDeputyEnable)
	time.Sleep(1 * time.Second)
	CheckMissingConnToOrders(networkOrders, nodeLeftNetwork)

	<-relievedOfDuty
	fmt.Println("Relieved of duty")
	transmitEnable <- false
	fmt.Println("Stopped transmitter")
	listenWranglerEnable <- false
	fmt.Println("Stopped glistenForWranglerConnections")
	sendOrderToDeputyEnable <- false
	fmt.Println("Stopped SendNodeOrdersToDeputy")
	quitAssigner <- true
	fmt.Println("Stopped Assigner")

}
func listenForWranglerConnections(
	incomingOrder chan<- Orderstatus,
	nodeLeftNetwork chan<- string,
	listenWranglerEnable <-chan bool) {

	ln, err := net.Listen("tcp", ":"+strconv.Itoa(config.TCP_port))
	if err != nil {
		fmt.Println("Error listening for connections:", err)
		return
	}

	newConn := make(chan net.Conn)
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				if errors.Is(err, net.ErrClosed) {
					return // if the listener is closed, return from the goroutine
				}
				fmt.Println("Error accepting connection:", err)
				continue
			}
			newConn <- conn
		}
	}()

	for {
		select {
		case enable := <-listenWranglerEnable:
			if !enable {
				fmt.Println("Stopping listenForWranglerConnections goroutine")
				ln.Close()
				return
			}
		case conn := <-newConn:
			reader := bufio.NewReader(conn)
			message, err := reader.ReadString('\n')
			if err != nil {
				fmt.Println("Error reading from connection while listeing for wranglers:", err)
				continue
			}

			peerID := strings.TrimSpace(message)

			WranglerConnections[peerID] = conn

			fmt.Println("Accepted Wrangler", peerID)
			fmt.Println(WranglerConnections)
			go ReceiveMessage(conn, incomingOrder, peerID, nodeLeftNetwork)
		}
	}
}

func SendNodeOrdersToDeputy(networkOrders *NetworkOrders, nodeOrdersUpdateChan chan bool, sendOrderToDeputyEnable <-chan bool) {
	ticker := time.NewTicker(DEPUTY_SEND_FREQ)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			SendDeputyMessage(networkOrders)
			//add updatechan

		case <-nodeOrdersUpdateChan:
			SendDeputyMessage(networkOrders)
			ticker.Reset(DEPUTY_SEND_FREQ)

		case enable := <-sendOrderToDeputyEnable:
			if !enable {
				fmt.Println("Stopping SendNodeOrdersToDeputy goroutine")
				return
			}
		}
	}
}

func SendDeputyMessage(networkOrders *NetworkOrders) {
	networkOrders.Mutex.Lock()
	defer networkOrders.Mutex.Unlock()
	var chosenOneID string
	for id, conn := range WranglerConnections {
		if chosenOneID == "" || WranglerConnections[chosenOneID] == nil {
			chosenOneID = id
		}
		nodeOrdersData := NetworkOrdersData{
			NetworkOrders: networkOrders.Orders,
			TheChosenOne:  id == chosenOneID, // or false, depending on your logic
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
		fmt.Println("Nodeorders:", networkOrders.Orders)
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

func ReceiveMessage(
	conn net.Conn,
	incomingOrder chan<- Orderstatus,
	peerID string,
	nodeLeftNetwork chan<- string) (Orderstatus, error) {
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		message := scanner.Text()

		// Convert the message from JSON to Orderstatus
		var order Orderstatus
		err := json.Unmarshal([]byte(message), &order)
		if err != nil {
			fmt.Println("Error unmarshalling order:", err)
			continue
		}

		fmt.Println("Received order from", peerID)

		//DeputyUpdateChan <- true

		incomingOrder <- order
	}

	fmt.Println("Error reading from connection:")
	fmt.Println("closing connection to", peerID)
	conn.Close()
	nodeLeftNetwork <- peerID
	delete(WranglerConnections, peerID)
	return Orderstatus{}, nil

}

func CloseConns(id string) {

	if WranglerConnections[id] != nil {
		fmt.Println("Closing connection to", id)
		WranglerConnections[id].Close()
	} else {
		fmt.Println("Connection already closed", id)
	}
}
