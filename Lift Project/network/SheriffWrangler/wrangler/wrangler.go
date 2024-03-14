package wrangler

import (
	"bufio"
	"encoding/json"
	"fmt"
	"mymodule/config"
	elevatorFSM "mymodule/elevator"
	"mymodule/elevator/elevio"
	"mymodule/network/conn"
	"mymodule/types"
	. "mymodule/types"
	"net"
	"time"
)

const N_FLOORS = config.N_FLOORS
const N_BUTTONS = config.N_BUTTONS

var sheriffConn net.Conn
var orderSent = make(chan Orderstatus)
var nodeOrdersReceived = make(chan NetworkOrdersData)

func ConnectWranglerToSheriff(sheriffIP string) bool {
	fmt.Println("netdial to sheriff:", sheriffIP)
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
	conn.SetReadDeadline(time.Now().Add(config.SHERIFF_IP_DEADLINE))
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

// ReceiveMessageFromsheriff receives an order from the sheriff and sends an acknowledgement
func ReceiveMessageFromSheriff(
	orderAssigned chan<- Order,
	sheriffDead chan<- NetworkOrdersData,
	requestSystemState chan<- bool,
	systemState <-chan map[string]Elev,
	addToLocalQueue chan<- Order) {

	var lastNetworkOrdersData NetworkOrdersData

	go acknowledger(orderSent, nodeOrdersReceived, requestSystemState, systemState, addToLocalQueue)

	scanner := bufio.NewScanner(sheriffConn)
	for scanner.Scan() {
		message := scanner.Text()
		fmt.Println("Received something")
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

			fmt.Println("Order received from sheriff:", order)
			orderAssigned <- Order{Floor: order.Floor, Button: order.Button}
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
			nodeOrdersReceived <- nodeOrdersData
			elevatorFSM.UpdateLightsFromNetworkOrders(nodeOrdersData.NetworkOrders)
			lastNetworkOrdersData = nodeOrdersData

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

func handleOrderSent(
	orderstatus Orderstatus,
	unacknowledgedButtons *[N_FLOORS][N_BUTTONS]bool,
	unacknowledgedComplete *[N_FLOORS][N_BUTTONS]bool,
	orderTickers *[N_FLOORS][N_BUTTONS]*time.Ticker,
	orderRetryCounts *[N_FLOORS][N_BUTTONS]int,
	quitChannels *[N_FLOORS][N_BUTTONS]chan bool,
) {
	if orderstatus.Served {
		unacknowledgedComplete[orderstatus.Floor][orderstatus.Button] = true
	} else {
		unacknowledgedButtons[orderstatus.Floor][orderstatus.Button] = true
	}

	ticker := time.NewTicker(500 * time.Millisecond)
	orderTickers[orderstatus.Floor][orderstatus.Button] = ticker

	quit := make(chan bool, 10)
	quitChannels[orderstatus.Floor][orderstatus.Button] = quit

	go resendOrder(orderstatus, ticker, quit, orderRetryCounts)
}

func resendOrder(
	order Orderstatus,
	ticker *time.Ticker,
	quit <-chan bool,
	orderRetryCounts *[N_FLOORS][N_BUTTONS]int,
) {
	const maxRetries = 5
	for {
		select {
		case <-ticker.C:
			if orderRetryCounts[order.Floor][order.Button] >= maxRetries {
				ticker.Stop()
				fmt.Println("Order withgiving iup")
				orderRetryCounts[order.Floor][order.Button] = 0
				return
			}
			fmt.Printf("Resending order on floor %d, button %d\n", order.Floor, order.Button)
			resendOrderToSheriff(order)
			orderRetryCounts[order.Floor][order.Button]++
		case <-quit:
			ticker.Stop()
			return
		}
	}
}

func handleNetworkOrders(
	networkorders NetworkOrdersData,
	unacknowledgedButtons *[N_FLOORS][N_BUTTONS]bool,
	unacknowledgedComplete *[N_FLOORS][N_BUTTONS]bool,
	quitChannels *[N_FLOORS][N_BUTTONS]chan bool,
) {
	fmt.Println("Received network orders from ", networkorders)

	for floor := 0; floor < N_FLOORS; floor++ {
		for button := 0; button < N_BUTTONS; button++ {
			if networkorders.NetworkOrders[floor][button] != "" {
				if unacknowledgedButtons[floor][button] {
					unacknowledgedButtons[floor][button] = false
					fmt.Println("Order acknowledged by sheriff:", floor, button)
					quitChannels[floor][button] <- true
				}
			} else if unacknowledgedComplete[floor][button] {
				fmt.Println("Order acknowledged by sheriff:", floor, button)
				unacknowledgedComplete[floor][button] = false
				quitChannels[floor][button] <- true
			}
		}
	}
}

func acknowledger(OrderSent <-chan Orderstatus, networkOrdersRecieved <-chan NetworkOrdersData, requestSystemState chan<- bool, systemState <-chan map[string]Elev, addToLocalQueue chan<- Order) {
	unacknowledgedButtons := [N_FLOORS][N_BUTTONS]bool{}
	unacknowledgedComplete := [N_FLOORS][N_BUTTONS]bool{}
	orderTickers := [N_FLOORS][N_BUTTONS]*time.Ticker{}
	orderRetryCounts := [N_FLOORS][N_BUTTONS]int{}
	quitChannels := [N_FLOORS][N_BUTTONS]chan bool{}

	for {
		select {
		case orderstatus := <-OrderSent:
			handleOrderSent(orderstatus, &unacknowledgedButtons, &unacknowledgedComplete, &orderTickers, &orderRetryCounts, &quitChannels)
		case networkorders := <-networkOrdersRecieved:
			handleNetworkOrders(networkorders, &unacknowledgedButtons, &unacknowledgedComplete, &quitChannels)
			checkSync(requestSystemState, systemState, networkorders.NetworkOrders, addToLocalQueue)
		}
	}
}

func checkSync(requestSystemState chan<- bool, systemState <-chan map[string]Elev, networkOrders [config.N_FLOORS][config.N_BUTTONS]string, addToLocalQueue chan<- Order) {

	for floor := 0; floor < config.N_FLOORS; floor++ {
		for button := 0; button < config.N_BUTTONS; button++ {
			if networkOrders[floor][button] != "" {
				requestSystemState <- true
				localSystemState := <-systemState
				assignedElev, existsInSystemState := localSystemState[networkOrders[floor][button]]
				if !existsInSystemState || !assignedElev.Queue[floor][button] {
					if networkOrders[floor][button] == config.Self_id {
						addToLocalQueue <- Order{Floor: floor, Button: elevio.ButtonType(button)}
						fmt.Println("WARNING - Order not in sync with system state, reassigning order TO MYSELF KJØH")
					}
				}
			}
		}
	}

}
