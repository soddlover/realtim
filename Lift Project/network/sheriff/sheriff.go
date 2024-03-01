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
	"time"
)

type Message struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

var WranglerConnections = make(map[string]net.Conn)

// var DeputyIDChan = make(chan string)
// var DeputyUpdateChan = make(chan bool)
var NewDeputyConnChan = make(chan net.TCPConn)
var DeputyDisconnectChan = make(chan net.TCPConn)
var NewWranglerConnChan = make(chan net.Conn)

const DEPUTY_SEND_FREQ = 3 * time.Second

func Sheriff(incomingOrder chan elev.Orderstatus, networkOrders map[string]elev.Orderstatus) {
	ipID := strings.Split(string(config.Self_id), ":")
	go peers.Transmitter(config.Sheriff_port, ipID[0], make(chan bool)) //channel for turning off sheriff transmitt?
	//go peers.Receiver(15647, peerUpdateCh)
	//go deputyUpdater(networkOrders)
	go deputyHandeler(networkOrders)
	go listenForWranglerConnections(incomingOrder)
	//go listenForDeputyConnection()

}

// func deputyUpdater(networkOrders map[string]elev.Orderstatus) {
// 	var deputyID string
// 	ticker := time.NewTicker(7 * time.Second)
// 	defer ticker.Stop()

// 	for {
// 		select {

// 		case newDeputyConn := <-NewDeputyConnChan:

// 		case <-DeputyUpdateChan:
// 			//check if deputy is in the list of connections
// 			if _, ok := Connections[deputyID]; !ok {
// 				deputyID = ChooseNewDeputy()
// 			}
// 			//send all orders to deputy
// 			SendDeputyMessage(deputyID, networkOrders)
// 		case <-ticker.C:
// 			//send all orders to deputy every 7 seconds
// 			SendDeputyMessage(deputyID, networkOrders)
// 		}
// 	}
// }

func deputyHandeler(nodeOrders map[string]elev.Orderstatus) {
	var deputyConn net.TCPConn
	var ticker *time.Ticker
	ticker = time.NewTicker(DEPUTY_SEND_FREQ)
	ticker.Stop()
	tickerRunning := false

	//deputyConn = <-NewDeputyConnChan

	for {
		select {
		case <-DeputyDisconnectChan:
			if tickerRunning {
				ticker.Stop()
				tickerRunning = false
			}
			if len(WranglerConnections) != 0 {
				wranglerConn, _ := ChooseNewDeputy()
				go initFirstDeputy(wranglerConn)
			} else {
				fmt.Println("No wrangler connections")
			}

		case deputyConn = <-NewDeputyConnChan:
			if !tickerRunning {
				ticker.Reset(DEPUTY_SEND_FREQ)
				tickerRunning = true
			}

		case <-ticker.C:
			fmt.Println("Sending node orders to deputy")
			go SendDeputyMessage(deputyConn, nodeOrders)
		}
	}
}

func initFirstDeputy(wranglerConn net.Conn) {
	sendRequestToBecomeDeputy(wranglerConn)
	go listenForDeputyConnection()
}

func sendRequestToBecomeDeputy(wranglerConn net.Conn) {

	msg := Message{
		Type: "requestToBecomeDeputy",
		Data: nil,
	}

	// Convert the message to JSON
	msgJSON, err := json.Marshal(msg)
	if err != nil {
		fmt.Println("Error marshalling deputy message:", err)
	}

	_, err = fmt.Fprintln(wranglerConn, string(msgJSON))
	// if err != nil {
	// 	fmt.Println("Error sending node orders to deputy, he might be dead:", err)

	// 	return false, err
	// }

	fmt.Println("Sent request to wrangler to become deputy")
	// return true, nil
}

func listenForWranglerConnections(incomingOrder chan elev.Orderstatus) {
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
		if _, ok := WranglerConnections[strings.Split(conn.RemoteAddr().String(), ":")[0]+":1"]; ok {
			peerID = strings.Split(conn.RemoteAddr().String(), ":")[0] + ":2"
			WranglerConnections[peerID] = conn
		} else {
			peerID = strings.Split(conn.RemoteAddr().String(), ":")[0] + ":1"
			WranglerConnections[peerID] = conn
		}

		fmt.Println("Accepted Wrangler", conn.RemoteAddr())
		// DeputyUpdateChan <- true
		go ReceiveMessage(conn, incomingOrder, peerID)

		//NewWranglerConnChan <- conn

		if len(WranglerConnections) == 1 {
			go initFirstDeputy(conn)
		}
	}
}

func listenForDeputyConnection() {
	ln, err := net.Listen("tcp", ":"+strconv.Itoa(config.Sheriff_deputy_port))
	if err != nil {
		fmt.Println("Error listening deputy connection:", err)
		return
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println("Error accepting deputy connection:", err)
			continue
		}

		tcpConn, ok := conn.(*net.TCPConn)
		if !ok {
			fmt.Println("Error casting to TCPConn")
			continue
		}

		err = tcpConn.SetKeepAlive(true)
		if err != nil {
			fmt.Println("Error setting keepalive:", err)
		}

		err = tcpConn.SetKeepAlivePeriod(10 * time.Minute)
		if err != nil {
			fmt.Println("Error setting keepalive period:", err)
		}

		fmt.Println("Accepted deputy connection from", conn.RemoteAddr())
		NewDeputyConnChan <- *tcpConn

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
	fmt.Println("Connections:", WranglerConnections)
	fmt.Println("Peer:", peer)
	conn, ok := WranglerConnections[peer]

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

func SendDeputyMessage(deputyConn net.TCPConn, nodeOrders map[string]elev.Orderstatus) (bool, error) {

	//conn, ok := Connections[peer]

	// if !ok {
	// 	fmt.Println("No connection to peer", peer)
	// 	return false, fmt.Errorf("no connection to peer %s", peer)
	// }

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

	_, err = fmt.Fprintln(&deputyConn, string(msgJSON))
	if err != nil {
		fmt.Println("Error sending node orders to deputy, he might be dead:", err)
		//deputyConn.Close()
		DeputyDisconnectChan <- deputyConn

		return false, err
	}

	fmt.Println("Sent node orders to deputy.")
	return true, nil

}

func ReceiveMessage(conn net.Conn, incomingOrder chan elev.Orderstatus, peerID string) (elev.Orderstatus, error) {
	for {
		reader := bufio.NewReader(conn)
		message, err := reader.ReadString('\n')
		if err != nil {

			if err.Error() == "EOF" {
				fmt.Println("Connection closed by", peerID)

				conn.Close()
				delete(WranglerConnections, peerID)

				//DeputyUpdateChan <- true
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

		if order.Status == false {
			incomingOrder <- order
		}
	}
}

func ChooseNewDeputy() (net.Conn, error) {
	//fmt.Println("Choosing new deputy")
	for k := range WranglerConnections {
		fmt.Println("The new deputy is:", k)
		return WranglerConnections[k], nil
	}
	fmt.Println("No wrangler connections!!!!")

	err := fmt.Errorf("no wrangler connections")
	fmt.Println(err)
	return nil, err
}
