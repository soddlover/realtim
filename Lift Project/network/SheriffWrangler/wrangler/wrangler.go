package wrangler

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	. "mymodule/config"
	"mymodule/network/conn"
	"mymodule/types"
	. "mymodule/types"
	"net"
	"time"
)

var sheriffConn net.Conn

func ConnectWranglerToSheriff(sheriffIP string) bool {

	fmt.Println("netdial to sheriff:", sheriffIP)
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", sheriffIP, TCP_PORT))

	if err != nil {
		fmt.Println("Error connecting to sheriff:", err)
		return false
	}

	fmt.Fprintf(conn, "%s\n", SELF_ID)
	fmt.Println("sent id to sheriff:", SELF_ID)
	sheriffConn = conn
	return true
}

func GetSheriffIP() string {

	var buf [1024]byte

	conn := conn.DialBroadcastUDP(SHERIFF_TRANSMITT_IP_PORT)
	defer conn.Close()
	conn.SetReadDeadline(time.Now().Add(SHERIFF_IP_DEADLINE))
	n, _, err := conn.ReadFrom(buf[0:])
	if err != nil {
		fmt.Println("Error reading from sheriff:", err)
		return ""
	}
	return string(buf[:n])

}

func SendOrderToSheriff(order Orderstatus) (bool, error) {

	orderJSON, err := json.Marshal(order)
	if err != nil {
		fmt.Println("Error marshalling order:", err)
		return false, err
	}

	_, err = fmt.Fprintln(sheriffConn, string(orderJSON))
	if err != nil {
		fmt.Println("Error sending order:", err)
		return false, err
	}

	fmt.Println("Order sent to sheriff:", order)
	return true, nil
}

// ReceiveMessageFromsheriff receives an order from the sheriff and sends an acknowledgement
func ReceiveTCPMessageFromSheriff(
	sheriffDead chan<- bool,
	addToLocalQueue chan<- Order) {

	scanner := bufio.NewScanner(sheriffConn)
	for scanner.Scan() {
		message := scanner.Text()
		//fmt.Println("Received message from sheriff:", message)

		var msg types.Message
		err := json.Unmarshal([]byte(message), &msg)
		if err != nil {
			fmt.Println("Error parsing message:", err)
			sheriffConn.Close()
			sheriffDead <- true
			continue
		}

		switch msg.Type {
		case "order":
			var order Orderstatus
			err = json.Unmarshal(msg.Data, &order)
			if err != nil {
				fmt.Println("Error parsing order:", err)
				continue
			}

			addToLocalQueue <- Order{Floor: order.Floor, Button: order.Button}
			// Send the order to the elevator

		default:
			fmt.Println("Unknown message type:", msg.Type)
		}
	}
	err := scanner.Err()
	fmt.Println("Error reading from sheriff as wrangler:", err)
	sheriffConn.Close()
	sheriffDead <- true

}
func ReceiveUDPNodeOrders(
	recievedNetworkOrders chan<- NetworkOrderPacket) {

	// Create a UDP address for the broadcast
	addr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("255.255.255.255:%d", UDP_NETWORK_ORDERS_PORT))
	if err != nil {
		log.Fatal(err)
	}

	// Listen for UDP broadcasts
	conn, err := net.ListenUDP("udp4", addr)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	buf := make([]byte, 1024)
	lastSequenceNumber := -1

	for {
		n, _, err := conn.ReadFromUDP(buf)
		if err != nil {
			fmt.Println("Error reading from UDP:", err)
			return
		}

		message := string(buf[:n])

		var msg types.Message
		err = json.Unmarshal([]byte(message), &msg)
		if err != nil {
			fmt.Println("Error parsing message:", err)
			continue
		}

		if msg.Type == "NodeOrders" {
			var nodeOrdersData NetworkOrderPacket
			err = json.Unmarshal(msg.Data, &nodeOrdersData)
			if err != nil {
				fmt.Println("Error parsing order:", err)
				continue
			}
			if (nodeOrdersData.SequenceNum > lastSequenceNumber) ||
				((lastSequenceNumber - nodeOrdersData.SequenceNum) > (SEQUENCE_NUMBER_LIMIT / 2)) {
				lastSequenceNumber = nodeOrdersData.SequenceNum
				recievedNetworkOrders <- nodeOrdersData
			}

		}
	}
}

func CloseSheriffConn() {

	err := sheriffConn.Close()
	if err != nil {
		fmt.Println("Error closing sheriff connection:", err)
		return
	}
	fmt.Println("Sheriff connection closed")
}

func CheckSync(
	requestSystemState chan<- bool,
	systemState <-chan map[string]Elev,
	networkOrders [N_FLOORS][N_BUTTONS]string,
	addToLocalQueue chan<- Order) {

	for floor, floorOrders := range networkOrders {
		for button, assignedID := range floorOrders {
			if assignedID != "" {
				requestSystemState <- true
				localSystemState := <-systemState
				assignedElev, existsInSystemState := localSystemState[assignedID]
				if !existsInSystemState || !assignedElev.Queue[floor][button] {
					if assignedID == SELF_ID {
						addToLocalQueue <- Order{Floor: floor, Button: ButtonType(button)}
					}
				}
			}
		}
	}

}
