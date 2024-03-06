package sheriff

import (
	"bufio"
	"encoding/json"
	"fmt"
	"mymodule/config"
	"mymodule/network/conn"
	. "mymodule/types"
	"net"
	"time"
)

var sheriffConn net.Conn

func ConnectWranglerToSheriff(sheriffIP string) bool {
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", sheriffIP, config.TCP_port))
	if err != nil {
		fmt.Println("Error connecting to sheriff:", err)
		return false
	}

	sheriffConn = conn
	return true
}

func GetSheriffIP() string {
	var buf [1024]byte

	conn := conn.DialBroadcastUDP(config.Sheriff_port)
	defer conn.Close()
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	n, _, err := conn.ReadFrom(buf[0:])
	if err != nil {
		fmt.Println("Error reading from sheriff:", err)
		return ""
	}
	return string(buf[:n])

}

// SendOrderToSheriff sends an order to the sheriff and waits for an acknowledgement
func SendOrderToSheriff(order Orderstatus) (bool, error) {
	// Convert the order to JSON

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
func ReceiveMessageFromSheriff(orderAssigned chan Orderstatus) (Orderstatus, error) {
	for {
		reader := bufio.NewReader(sheriffConn)
		message, err := reader.ReadString('\n')
		//fmt.Println("Received message from sheriff:", message)
		if err != nil {
			fmt.Println("Error reading from sheriff:", err)
			time.Sleep(5 * time.Second)
			continue
		}

		var msg Message
		err = json.Unmarshal([]byte(message), &msg)
		if err != nil {
			fmt.Println("Error parsing message:", err)
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

			fmt.Println("Order received from sheriff:", order)
			orderAssigned <- order // Send the order to the elevator

		case "requestToBecomeDeputy":
			fmt.Println("Received request to become deputy from sheriff")
			initDeputy() //not sure if it should be go'ed or not

		default:
			fmt.Println("Unknown message type:", msg.Type)
		}
	}
}
