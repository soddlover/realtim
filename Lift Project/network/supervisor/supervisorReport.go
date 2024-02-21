package supervisor

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

type OrderInfo struct {
	Completed  bool
	Floor      int
	ButtonType elevatorFSM.ButtonType //Hvorfor er dette feil?
}

var superVisorConn net.Conn
var ordersMap = make(map[string]OrderInfo) // string is used for the order ID, dårlig navn i guess

// ConnectToSupervisor connects to the supervisor and returns the connection
func ConnectToSupervisor(supervisorIP string) bool {
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", supervisorIP, config.TCP_port))
	if err != nil {
		fmt.Println("Error connecting to supervisor:", err)
		return false
	}
	superVisorConn = conn
	return true
}

func GetSupervisorIP() string {
	var buf [1024]byte

	conn := conn.DialBroadcastUDP(config.Supervisor_port)
	defer conn.Close()
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	n, _, err := conn.ReadFrom(buf[0:])
	if err != nil {
		fmt.Println("Error reading from supervisor:", err)
		return ""
	}
	return string(buf[:n])

}

// SendOrderToSupervisor sends an order to the supervisor and waits for an acknowledgement
func SendOrderToSupervisor(order Orderstatus) (bool, error) {
	// Convert the order to JSON
	orderJSON, err := json.Marshal(order)
	if err != nil {
		fmt.Println("Error marshalling order:", err)
		return false, err
	}

	_, err = fmt.Fprintln(superVisorConn, string(orderJSON))
	if err != nil {
		fmt.Println("Error sending order:", err)
		return false, err
	}

	// Wait for an acknowledgement
	// superVisorConn.SetReadDeadline(time.Now().Add(5 * time.Second))
	// reader := bufio.NewReader(superVisorConn)
	// acknowledgement, err := reader.ReadString('\n')
	// superVisorConn.SetReadDeadline(time.Time{})
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
	fmt.Println("Order sent to supervisor:", order)
	return true, nil
}

// ReceiveOrderFromSupervisor receives an order from the supervisor and sends an acknowledgement
func ReceiveOrderFromSupervisor(orderAssigned chan elevatorFSM.Order) (Orderstatus, error) {
	for {
		reader := bufio.NewReader(superVisorConn)
		message, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading from supervisor:", err)
			time.Sleep(5 * time.Second)
			continue
		}

		// Convert the message from JSON to Orderstatus
		var order Orderstatus
		err = json.Unmarshal([]byte(message), &order)
		if err != nil {
			fmt.Println("Error unmarshalling order:", err)
			continue
		}

		// // Send an acknowledgement
		// _, err = fmt.Fprintln(superVisorConn, "ACK")
		// if err != nil {
		// 	fmt.Println("Error sending acknowledgement:", err)
		// 	continue
		// }

		fmt.Println("Order received from supervisor:", order)
		orderAssigned <- elevatorFSM.Order{Floor: order.Floor, Button: order.Button} // Send the order to the elevator
		ordersMap[order.OrderID] = OrderInfo{
			Completed:  false,
			Floor:      order.Floor,
			ButtonType: order.Button,
		}
	}
}

func MarkOrderAsCompleted(orderID string) {
	if orderInfo, exists := ordersMap[orderID]; exists {
		orderInfo.Completed = true
		ordersMap[orderID] = orderInfo
	}
}
