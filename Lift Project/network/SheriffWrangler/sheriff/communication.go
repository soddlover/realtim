package sheriff

import (
	. "Project/config"
	"Project/network/conn"
	. "Project/types"
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

const IP_TRANSMITT_INTERVAL = 15 * time.Millisecond

var deputyID string
var wranglerConnections = make(map[string]net.Conn)
var udpConn net.PacketConn
var seqNum int

func transmittIP() {
	ip := strings.Split(string(SELF_ID), ":")[0]
	conn := conn.DialBroadcastUDP(SHERIFF_TRANSMITT_IP_PORT)
	addr, _ := net.ResolveUDPAddr("udp4", fmt.Sprintf("255.255.255.255:%d", SHERIFF_TRANSMITT_IP_PORT))
	for {
		<-time.After(IP_TRANSMITT_INTERVAL)
		conn.WriteTo([]byte(ip), addr)
	}
}

func EstablishWranglerCommunications(
	assignOrder chan<- Orderstatus,
	nodeUnavailabe chan<- string,
	latestSequenceNr int) {

	go transmittIP()
	udpConn = conn.DialBroadcastUDP(UDP_NETWORK_ORDERS_PORT)

	if latestSequenceNr > 0 {
		seqNum = latestSequenceNr
	}

	ln, err := net.Listen("tcp", ":"+strconv.Itoa(TCP_PORT))
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
					return
				}
				fmt.Println("Error accepting connection:", err)
				continue
			}
			newConn <- conn
		}
	}()

	for {
		conn := <-newConn
		fmt.Println("Incoming new connection waiting to recieve ID")
		reader := bufio.NewReader(conn)
		done := make(chan bool)

		go func() {
			message, err := reader.ReadString('\n')
			if err != nil {
				fmt.Println("Error reading from connection:", err)
				conn.Close()
				done <- false
			}

			peerID := strings.TrimSpace(message)
			wranglerConnections[peerID] = conn
			fmt.Printf("ID recieved: %s\n", peerID)
			go ReceiveMessage(conn, assignOrder, peerID, nodeUnavailabe)
			done <- true
		}()

		select {
		case success := <-done:
			if success {
				fmt.Println("Connections: ", wranglerConnections)
			}

		case <-time.After(TCP_ESTABLISH_DEADLINE):
			fmt.Println("Timeout reading from connection, closing it")
			conn.Close()
		}
	}
}

func SendNetworkOrders(networkOrders [N_FLOORS][N_BUTTONS]string) {

	addr, _ := net.ResolveUDPAddr("udp4", fmt.Sprintf("255.255.255.255:%d", UDP_NETWORK_ORDERS_PORT))

	for id := range wranglerConnections {
		if deputyID == "" || wranglerConnections[deputyID] == nil {
			deputyID = id
		}
	}

	seqNum = (seqNum + 1) % SEQUENCE_NUMBER_LIMIT

	nodeOrdersData := NetworkOrderPacket{
		Orders:      networkOrders,
		DeputyID:    deputyID,
		SequenceNum: seqNum, // or false, depending on your logic
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

	_, _ = udpConn.WriteTo(msgJSON, addr)
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
		fmt.Println("Error marshalling order:", err)
		return false, err
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
		var order Orderstatus
		err := json.Unmarshal([]byte(message), &order)

		if err != nil {
			fmt.Println("Error unmarshalling order:", err)
			continue
		}
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

func CloseAllConnections() {
	for id, conn := range wranglerConnections {
		fmt.Println("Closing connection to", id)
		conn.Close()
	}
}
