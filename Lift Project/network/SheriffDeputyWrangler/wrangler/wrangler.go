package wrangler

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"mymodule/config"
	"mymodule/network/conn"
	"mymodule/types"
	. "mymodule/types"
	"net"
	"time"
)

var sheriffConn net.Conn
var SheriffDisconnectedFromWrangler = make(chan bool)
var WranglerPromotion = make(chan bool)

func ConnectWranglerToSheriff(sheriffIP string) bool {
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", sheriffIP, config.TCP_port))
	if err != nil {
		fmt.Println("Error connecting to sheriff:", err)
		return false
	}
	fmt.Fprintf(conn, "%s\n", config.Self_id)
	fmt.Println("sent id to sheriff:", config.Self_id)
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
func ReceiveMessageFromSheriff(orderAssigned chan Orderstatus, networkchannels NetworkChannels) {
	var nodeOrdersData NodeOrdersData
	for {
		select {

		default:
			reader := bufio.NewReader(sheriffConn)
			message, err := reader.ReadString('\n')
			//fmt.Println("Received message from sheriff:", message)
			if err != nil {
				if err == io.EOF {
					fmt.Println("Connection closed by sheriff in wrangler")
					sheriffConn.Close()
					networkchannels.SheriffDead <- nodeOrdersData
					fmt.Println("we did it")
					return
				}
				fmt.Println("Error reading from sheriff as wrangles\r:", err)

			}

			var msg types.Message
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

			case "NodeOrders":
				//initDeputy() //not sure if it should be go'ed or not
				err = json.Unmarshal(msg.Data, &nodeOrdersData)
				if err != nil {
					fmt.Println("Error parsing order:", err)
					continue
				}
				fmt.Println("Received nodeOrdersData from sheriff:")

			default:
				fmt.Println("Unknown message type:", msg.Type)
			}
		}

	}
	//over 3 readerrors
}
