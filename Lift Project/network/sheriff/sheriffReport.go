package sheriff

import (
	"bufio"
	"encoding/json"
	"fmt"
	"mymodule/config"
	elevatorFSM "mymodule/elevator"
	"mymodule/network/conn"
	"net"
	"time"
)

var sheriffConn net.Conn

// ConnectToSheriff connects to the sheriff and returns the connection
func ConnectToSheriff(sheriffIP string) bool {
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
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
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

	// Wait for an acknowledgement
	// sheriffConn.SetReadDeadline(time.Now().Add(5 * time.Second))
	// reader := bufio.NewReader(sheriffConn)
	// acknowledgement, err := reader.ReadString('\n')
	// sheriffConn.SetReadDeadline(time.Time{})
	// if err != nil {
	// 	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
	// 		return false, fmt.Errorf("acknowledgement timed out")
	// 	}
	// 	return false, err
	// }

	// // Check the acknowledgement
	// if strings.TrimSpace(acknowledgement) != "ACK" {
	// 	return false, fmt.Errorf("unexpected acknowledgement: %s", acknowledgement)
	// }
	fmt.Println("Order sent to sheriff:", order)
	return true, nil
}

// ReceiveMessageFromsheriff receives an order from the sheriff and sends an acknowledgement
func ReceiveMessageFromSheriff(orderAssigned chan elevatorFSM.Order) (Orderstatus, error) {
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
			orderAssigned <- elevatorFSM.Order{Floor: order.Floor, Button: order.Button} // Send the order to the elevator

		case "deputyMessage":
			var deputyNodeOrders map[string]Orderstatus
			err = json.Unmarshal(msg.Data, &deputyNodeOrders)
			if err != nil {
				fmt.Println("Error parsing deputy message:", err)
				continue
			}
			// Handle deputy message...
			fmt.Println("Received deputy message from sheriff")

		default:
			fmt.Println("Unknown message type:", msg.Type)
		}

		// // Send an acknowledgement
		// _, err = fmt.Fprintln(sheriffConn, "ACK")
		// if err != nil {
		// 	fmt.Println("Error sending acknowledgement:", err)
		// 	continue
		// }

	}
}
