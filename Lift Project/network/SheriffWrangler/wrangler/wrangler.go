package wrangler

import (
	"bufio"
	"encoding/json"
	"fmt"
	"mymodule/config"
	"mymodule/elevator/elevio"
	"mymodule/network/conn"
	"mymodule/types"
	. "mymodule/types"
	"net"
	"os"
	"time"
)

const N_FLOORS = config.N_FLOORS
const N_BUTTONS = config.N_BUTTONS

var sheriffConn net.Conn
var orderSent = make(chan Orderstatus)
var nodeOrdersReceived = make(chan NetworkOrdersData)

func ConnectWranglerToSheriff(sheriffIP string) bool {
	fmt.Println("netdial to sheriff:", sheriffIP)
	timeout := time.Duration(15 * time.Second)
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", sheriffIP, config.TCP_port), timeout)

	if err != nil {
		fmt.Println("Error connecting to sheriff:", err)
		conn.Close()
		time.Sleep(5 * time.Second)
		os.Exit(1)
		return false
	}

	fmt.Fprintf(conn, "%s\n", config.Id)
	fmt.Println("sent id to sheriff:", config.Id)
	sheriffConn = conn
	return true
}

func GetSheriffIP() string {
	var buf [1024]byte

	conn := conn.DialBroadcastUDP(config.Sheriff_port)
	defer conn.Close()
	conn.SetReadDeadline(time.Now().Add(config.SHERIFF_IP_DEADLINE))
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
func ReceiveMessageFromSheriff(
	sheriffDead chan<- NetworkOrdersData,
	recievedNetworkOrders chan<- NetworkOrdersData,
	addToLocalQueue chan<- Order) {

	var lastNetworkOrdersData NetworkOrdersData

	scanner := bufio.NewScanner(sheriffConn)
	for scanner.Scan() {
		message := scanner.Text()
		//fmt.Println("Received message from sheriff:", message)

		var msg types.Message
		err := json.Unmarshal([]byte(message), &msg)
		if err != nil {
			fmt.Println("Error parsing message:", err)
			sheriffConn.Close()
			sheriffDead <- lastNetworkOrdersData
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
			fmt.Println("Deadlocking addtolocalquee?")
			addToLocalQueue <- Order{Floor: order.Floor, Button: order.Button}
			fmt.Println("NO Deadlocking addtolocalquee?")
			// Send the order to the elevator

		case "NodeOrders":
			var nodeOrdersData NetworkOrdersData

			//initDeputy() //not sure if it should be go'ed or not
			err = json.Unmarshal(msg.Data, &nodeOrdersData)
			if err != nil {
				fmt.Println("Error parsing order:", err)
				continue
			}
			fmt.Println("Deadlocking recievednetworkordeers?")
			fmt.Println(nodeOrdersData)
			lastNetworkOrdersData = nodeOrdersData
			recievedNetworkOrders <- nodeOrdersData
			fmt.Println("NO Deadlocking recievednetworkordeers?")
			fmt.Println("*******")
			fmt.Println("")

		default:
			fmt.Println("Unknown message type:", msg.Type)
		}
	}
	err := scanner.Err()
	fmt.Println("Error reading from sheriff as wrangler:", err)
	sheriffConn.Close()
	sheriffDead <- lastNetworkOrdersData
	return

}
func CloseSheriffConn() {
	err := sheriffConn.Close()
	if err != nil {
		fmt.Println("Error closing sheriff connection:", err)
		return
	}
	fmt.Println("Sheriff connection closed")
}

func CheckSync(requestSystemState chan<- bool, systemState <-chan map[string]Elev, networkOrders [config.N_FLOORS][config.N_BUTTONS]string, addToLocalQueue chan<- Order) {

	for floor := 0; floor < config.N_FLOORS; floor++ {
		for button := 0; button < config.N_BUTTONS; button++ {
			if networkOrders[floor][button] != "" {
				requestSystemState <- true
				localSystemState := <-systemState
				assignedElev, existsInSystemState := localSystemState[networkOrders[floor][button]]
				if !existsInSystemState || !assignedElev.Queue[floor][button] {
					if networkOrders[floor][button] == config.Id {
						addToLocalQueue <- Order{Floor: floor, Button: elevio.ButtonType(button)}
					}
				}
			}
		}
	}

}
