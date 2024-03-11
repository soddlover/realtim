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
var orderSent = make(chan Orderstatus)
var NodeOrdersReceived = make(chan NetworkOrdersData)

func ConnectWranglerToSheriff(sheriffIP string) bool {
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", sheriffIP, config.TCP_port))
	if err != nil {
		fmt.Println("Error connecting to sheriff:", err)
		return false
	}

	tcpConn, ok := conn.(*net.TCPConn)
	if !ok {
		fmt.Println("Error asserting connection type")
		return false
	}

	if err := tcpConn.SetKeepAlive(true); err != nil {
		fmt.Println("Error setting keepalive:", err)
		return false
	}

	// Set the keepalive period to 1 minute.
	if err := tcpConn.SetKeepAlivePeriod(3 * time.Second); err != nil {
		fmt.Println("Error setting keepalive period:", err)
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
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	n, _, err := conn.ReadFrom(buf[0:])
	if err != nil {
		fmt.Println("Error reading from sheriff:", err)
		return ""
	}
	return string(buf[:n])

}
func SendOrderToSheriff(order Orderstatus) (bool, error) {
	return sendOrderToSheriff(order, orderSent)
}

func resendOrderToSheriff(order Orderstatus) (bool, error) {
	return sendOrderToSheriff(order, nil)
}

func sendOrderToSheriff(order Orderstatus, OrderSent chan Orderstatus) (bool, error) {
	if OrderSent != nil {
		//OrderSent <- order
	}
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

// THIS FUNCTION DOES NOT WORK AT ALL YET BEWARE WTF!!!!!!!!
func acknowledger(OrderSent <-chan Orderstatus, networkOrdersRecieved <-chan NetworkOrdersData) {
	unacknowledgedButtons := [config.N_FLOORS][config.N_BUTTONS]bool{}
	unacknowledgedComplete := [config.N_FLOORS][config.N_BUTTONS]bool{}
	orderTickers := [config.N_FLOORS][config.N_BUTTONS]*time.Ticker{}
	orderRetryCounts := [config.N_FLOORS][config.N_BUTTONS]int{}
	const maxRetries = 5
	for {
		select {
		case orderstatus := <-OrderSent:
			if orderstatus.Served {
				unacknowledgedComplete[orderstatus.Floor][orderstatus.Button] = true
			} else {
				unacknowledgedButtons[orderstatus.Floor][orderstatus.Button] = true
			}
			ticker := time.NewTicker(time.Second * 1) // Resend the order every 10 seconds
			orderTickers[orderstatus.Floor][orderstatus.Button] = ticker
			go func(order Orderstatus) {
				for range ticker.C {
					if orderRetryCounts[order.Floor][order.Button] >= maxRetries {
						ticker.Stop()
						fmt.Printf("***************************************************************************************************")
						fmt.Printf("Order withgiving iup")
						fmt.Printf("***************************************************************************************************")
						orderRetryCounts[order.Floor][order.Button] = 0
						return
					}
					fmt.Printf("***************************************************************************************************")
					fmt.Printf("Resending order on floor %d, button %d\n", order.Floor, order.Button)
					fmt.Printf("***************************************************************************************************")

					resendOrderToSheriff(order)
					orderRetryCounts[order.Floor][order.Button]++
				}
			}(orderstatus)

		case networkorders := <-networkOrdersRecieved:
			// Which orders have been deleted?
			for floor := 0; floor < config.N_FLOORS; floor++ {
				for button := 0; button < config.N_BUTTONS; button++ {
					if networkorders.NetworkOrders[floor][button] != "" {
						unacknowledgedButtons[floor][button] = false
						ticker := orderTickers[floor][button]
						ticker.Stop()
						orderTickers[floor][button] = nil

					} else {
						unacknowledgedComplete[floor][button] = false
						ticker := orderTickers[floor][button]
						ticker.Stop()
						orderTickers[floor][button] = nil

					}
				}
			}
		}
	}
}

// ReceiveMessageFromsheriff receives an order from the sheriff and sends an acknowledgement
func ReceiveMessageFromSheriff(
	orderAssigned chan<- Orderstatus,
	sheriffDead chan<- NetworkOrdersData) {

	var lastnodeOrdersData NetworkOrdersData

	go acknowledger(orderSent, NodeOrdersReceived)
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
					sheriffDead <- lastnodeOrdersData
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
				orderAssigned <- order
				// Send the order to the elevator

			case "NodeOrders":
				var nodeOrdersData NetworkOrdersData

				//initDeputy() //not sure if it should be go'ed or not
				err = json.Unmarshal(msg.Data, &nodeOrdersData)
				if err != nil {
					fmt.Println("Error parsing order:", err)
					continue
				}
				fmt.Println("Received nodeOrdersData from sheriff:")
				NodeOrdersReceived <- nodeOrdersData
				elevatorFSM.UpdateLightsFromNetworkOrders(nodeOrdersData.NetworkOrders)
				lastnodeOrdersData = nodeOrdersData

			default:
				fmt.Println("Unknown message type:", msg.Type)
			}
		}

	}
	//over 3 readerrors
}
