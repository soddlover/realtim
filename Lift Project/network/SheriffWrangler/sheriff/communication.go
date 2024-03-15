package sheriff

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"mymodule/config"
	"mymodule/network/conn"
	. "mymodule/types"
	"net"
	"strconv"
	"strings"
	"time"
)

var chosenOneID string
var wranglerConnections = make(map[string]net.Conn)
var udpConn net.PacketConn
var seqNum int

func broadCastNetwork(seq int) {

	if seq > 0 {
		seqNum = seq
	}
	var err error
	udpConn = conn.DialBroadcastUDP(12345)
	if err != nil {
		log.Fatal(err)
	}

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

		wranglerConnections[peerID] = conn

		fmt.Println("Accepted Wrangler", peerID)
		fmt.Println(wranglerConnections)
		go ReceiveMessage(conn, assignOrder, peerID, nodeUnavailabe)

	}
}

func SendNetworkOrders(networkOrders [config.N_FLOORS][config.N_BUTTONS]string) {

	addr, _ := net.ResolveUDPAddr("udp4", fmt.Sprintf("255.255.255.255:%d", 12345))

	for id, _ := range wranglerConnections {
		if chosenOneID == "" || wranglerConnections[chosenOneID] == nil {
			chosenOneID = id
		}
	}
	nodeOrdersData := NetworkOrderPacket{
		NetworkOrders: networkOrders,
		TheChosenOne:  chosenOneID,
		SequenceNum:   seqNum, // or false, depending on your logic
	}
	seqNum++
	nodeOrdersDataJSON, err := json.Marshal(nodeOrdersData)
	if err != nil {
		fmt.Println("Error marshalling node orders to be sent to deputy:", err)
	}
	fmt.Println("Sequence number:", seqNum)

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

	_, err = udpConn.WriteTo(msgJSON, addr)
	if err != nil {
		fmt.Println("Error sending node orders to deputy, he might be dead:", err)
	}

}

func SendOrderMessage(peer string, order Orderstatus) (bool, error) {

	conn, ok := wranglerConnections[peer]
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
		delete(wranglerConnections, peer)
		return false, err
	}
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

		//DeputyUpdateChan <- true

		assignOrder <- order
	}

	fmt.Println("Error reading from connection:")
	fmt.Println("closing connection to", peerID)
	conn.Close()
	nodeUnavailabe <- peerID
	delete(wranglerConnections, peerID)
	return Orderstatus{}, nil
}

func CloseConns(id string) {

	if id == "ALL" {
		for id, conn := range wranglerConnections {
			fmt.Println("Closing connection to", id)
			conn.Close()
		}
	}
	if wranglerConnections[id] != nil {
		fmt.Println("Closing connection to", id)
		wranglerConnections[id].Close()
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
