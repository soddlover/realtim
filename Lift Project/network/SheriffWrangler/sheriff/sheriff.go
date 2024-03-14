package sheriff

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"mymodule/config"
	"mymodule/network/conn"
	. "mymodule/types"
	"net"
	"strconv"
	"strings"
	"time"
)

const DEPUTY_SEND_FREQ = 3 * time.Second

var chosenOneID string
var WranglerConnections = make(map[string]net.Conn)

func CheckMissingConnToOrders(
	networkOrders [config.N_FLOORS][config.N_BUTTONS]string,
	nodeUnavailabe chan<- string) {

	processedIDs := make(map[string]bool)

	fmt.Println("Checking for missing connections to orders")

	for floor := 0; floor < config.N_FLOORS; floor++ {
		for button := 0; button < config.N_BUTTONS; button++ {
			id := networkOrders[floor][button]
			//fmt.Printf("Checking order at floor %d, button %d, id: %s\n", floor, button, id) // Print the current order being checked
			if id != "" && WranglerConnections[id] == nil && id != config.Self_id && !processedIDs[id] {
				nodeUnavailabe <- id
				fmt.Println("***Missing connection to ACTIVE ORDER Reassigning order!!!***", id)
				processedIDs[id] = true
			} else {
				//fmt.Printf("Order at floor %d, button %d is not missing connection\n", floor, button) // Print a message if the order is not missing connection
			}
		}
	}
}

func Sheriff(
	assignOrder chan<- Orderstatus,
	requestNetworkOrders chan<- bool,
	networkorders <-chan [config.N_FLOORS][config.N_BUTTONS]string,
	nodeUnavailabe chan<- string) {

	ip := strings.Split(string(config.Self_id), ":")[0]
	go Transmitter(config.Sheriff_port, ip)
	go listenForWranglerConnections(assignOrder, nodeUnavailabe)
	time.Sleep(1 * time.Second)
	requestNetworkOrders <- true
	networkOrders := <-networkorders
	CheckMissingConnToOrders(networkOrders, nodeUnavailabe)
}

func listenForWranglerConnections(
	assignOrder chan<- Orderstatus,
	nodeUnavailabe chan<- string) {

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
		conn := <-newConn
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
		go ReceiveMessage(conn, assignOrder, peerID, nodeUnavailabe)

	}
}

func SendNetworkOrders(networkOrders [config.N_FLOORS][config.N_BUTTONS]string) {

	for id, conn := range WranglerConnections {
		if chosenOneID == "" || WranglerConnections[chosenOneID] == nil {
			chosenOneID = id
		}
		nodeOrdersData := NetworkOrdersData{
			NetworkOrders: networkOrders,
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
	}
}

func SendOrderMessage(peer string, order Orderstatus) (bool, error) {

	conn, ok := WranglerConnections[peer]
	if !ok {
		fmt.Println("No connection to ", peer)

		return false, fmt.Errorf("no connection to peer %s", peer)
	}
	orderJSON, err := json.Marshal(order)
	if err != nil {
		fmt.Println("Error marshalling order:", err)
		return false, err
	}
	msg := Message{
		Type: "order",
		Data: orderJSON,
	}
	msgJSON, err := json.Marshal(msg)
	if err != nil {
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
	assignOrder chan<- Orderstatus,
	peerID string,
	nodeUnavailabe chan<- string) (Orderstatus, error) {
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

		assignOrder <- order
	}

	fmt.Println("Error reading from connection:")
	fmt.Println("closing connection to", peerID)
	conn.Close()
	nodeUnavailabe <- peerID
	delete(WranglerConnections, peerID)
	return Orderstatus{}, nil
}

func CloseConns(id string) {
	if id == "ALL" {
		for id, conn := range WranglerConnections {
			fmt.Println("Closing connection to", id)
			conn.Close()
		}
	}
	if WranglerConnections[id] != nil {
		fmt.Println("Closing connection to", id)
		WranglerConnections[id].Close()
	} else {
		fmt.Println("Connection already closed", id)
	}
}

const interval = 15 * time.Millisecond

func Transmitter(port int, id string) {
	conn := conn.DialBroadcastUDP(port)
	addr, _ := net.ResolveUDPAddr("udp4", fmt.Sprintf("255.255.255.255:%d", port))
	for {
		<-time.After(interval)
		conn.WriteTo([]byte(id), addr)
	}
}
