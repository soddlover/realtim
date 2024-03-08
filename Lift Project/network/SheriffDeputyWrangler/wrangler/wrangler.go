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
var orderSent = make(chan Orderstatus)
var NodeOrdersReceived = make(chan NodeOrdersData)

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
func SendOrderToSheriff(order Orderstatus) (bool, error) {
	return sendOrderToSheriff(order, orderSent)
}

func resendOrderToSheriff(order Orderstatus) (bool, error) {
	return sendOrderToSheriff(order, nil)
}

func sendOrderToSheriff(order Orderstatus, OrderSent chan Orderstatus) (bool, error) {
	if OrderSent != nil {
		OrderSent <- order
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
func acknowledger(OrderSent chan Orderstatus, Nodeordersreceived chan NodeOrdersData) {
	unacknowledgedButtons := make(map[string]Orderstatus)
	unacknowledgedComplete := make(map[string]Orderstatus)
	orderTickers := make(map[string]*time.Ticker)
	orderRetryCounts := make(map[string]int)
	const maxRetries = 5

	var oldNodeOrdersData NodeOrdersData
	for {
		select {
		case orderstatus := <-OrderSent:
			if orderstatus.Status {
				unacknowledgedComplete[orderstatus.OrderID] = orderstatus
			} else {
				unacknowledgedButtons[orderstatus.OrderID] = orderstatus
			}
			ticker := time.NewTicker(time.Second * 1) // Resend the order every 10 seconds
			orderTickers[orderstatus.OrderID] = ticker
			go func(order Orderstatus) {
				for range ticker.C {
					if orderRetryCounts[order.OrderID] >= maxRetries {
						ticker.Stop()
						fmt.Printf("***************************************************************************************************")
						fmt.Printf("Order with id %s has been retried %d times and will not be retried again\n", order.OrderID, maxRetries)
						fmt.Printf("***************************************************************************************************")

						return
					}
					fmt.Printf("Resending order with id %s\n", order.OrderID)
					resendOrderToSheriff(order)
					orderRetryCounts[order.OrderID]++
				}
			}(orderstatus)

		case Nodeorders := <-Nodeordersreceived:
			// Which orders have been deleted?

			for id, _ := range oldNodeOrdersData.NodeOrders {
				if _, exists := Nodeorders.NodeOrders[id]; !exists {
					fmt.Printf("Order with id %s has been deleted\n", id)
					delete(unacknowledgedComplete, id)
					if ticker, ok := orderTickers[id]; ok {
						ticker.Stop()
						delete(orderTickers, id)
					}
				}
			}

			// Which orders have been added?
			for id, _ := range Nodeorders.NodeOrders {
				if _, exists := oldNodeOrdersData.NodeOrders[id]; !exists {
					fmt.Printf("Order with id %s has been added\n", id)
					delete(unacknowledgedButtons, id)
					if ticker, ok := orderTickers[id]; ok {
						ticker.Stop()
						delete(orderTickers, id)
					}
				}
			}

			oldNodeOrdersData = NodeOrdersData{
				NodeOrders:   make(map[string]Orderstatus),
				TheChosenOne: Nodeorders.TheChosenOne,
			}

			for k, v := range Nodeorders.NodeOrders {
				oldNodeOrdersData.NodeOrders[k] = Orderstatus{
					OrderID: v.OrderID,
					Owner:   v.Owner,
					Floor:   v.Floor,
					Button:  v.Button,
					Status:  v.Status,
				}
			}
		}
	}
}

// ReceiveMessageFromsheriff receives an order from the sheriff and sends an acknowledgement
func ReceiveMessageFromSheriff(orderAssigned chan Orderstatus, networkchannels NetworkChannels) {
	var lastnodeOrdersData NodeOrdersData

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
					networkchannels.SheriffDead <- lastnodeOrdersData
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
				var nodeOrdersData NodeOrdersData

				//initDeputy() //not sure if it should be go'ed or not
				err = json.Unmarshal(msg.Data, &nodeOrdersData)
				if err != nil {
					fmt.Println("Error parsing order:", err)
					continue
				}
				fmt.Println("Received nodeOrdersData from sheriff:")
				NodeOrdersReceived <- nodeOrdersData
				lastnodeOrdersData = nodeOrdersData

			default:
				fmt.Println("Unknown message type:", msg.Type)
			}
		}

	}
	//over 3 readerrors
}
