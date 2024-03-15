package sheriff

import (
	"fmt"
	"mymodule/config"
	. "mymodule/config"
	"mymodule/elevator"
	. "mymodule/types"
	"strings"
	"time"
)

func Sheriff(
	assignOrder chan Orderstatus,
	lastnetworkOrdersData NetworkOrderPacket,
	addToLocalQueue chan<- Order,
	requestSystemState chan<- bool,
	systemState <-chan map[string]Elev) {

	fmt.Println("Sheriff started with latest network order data: ", lastnetworkOrdersData)
	writeNetworkOrders := make(chan OrderID)
	nodeUnavailabe := make(chan string)
	networkorders := make(chan [N_FLOORS][N_BUTTONS]string)
	requestNetworkOrders := make(chan bool)

	ip := strings.Split(string(SELF_ID), ":")[0]
	go broadCastNetwork(
		lastnetworkOrdersData.SequenceNum)
	go Transmitter(
		config.Sheriff_port,
		ip)

	go listenForWranglerConnections(
		assignOrder,
		nodeUnavailabe)

	go Assigner(
		addToLocalQueue,
		requestSystemState,
		systemState,
		assignOrder,
		writeNetworkOrders,
		requestNetworkOrders,
		networkorders)

	go redistributor(
		nodeUnavailabe,
		assignOrder,	//requestSystemState chan<- bool,
		//systemState <-chan map[string]Elev,
		requestNetworkOrders,
		networkorders)

	go netWorkOrderHandler(
		requestNetworkOrders,
		writeNetworkOrders,
		networkorders,
		assignOrder,
		lastnetworkOrdersData.NetworkOrders)

	go checkForUnavailable(
		requestSystemState,
		systemState,
		nodeUnavailabe)

	time.Sleep(1 * time.Second)
	requestNetworkOrders <- true
	networkOrders := <-networkorders
	CheckMissingConnToOrders(networkOrders, nodeUnavailabe)
}

func netWorkOrderHandler(
	requestNetworkOrders <-chan bool,
	writeNetworkOrders <-chan OrderID,
	networkorders chan<- [config.N_FLOORS][config.N_BUTTONS]string,
	assignOrder chan<- Orderstatus,
	lastNetworkOrders [config.N_FLOORS][config.N_BUTTONS]string) {

	NetworkOrders := lastNetworkOrders
	orderTimestamps := [config.N_FLOORS][config.N_BUTTONS]time.Time{}
	ticker := time.NewTicker(NETWORK_ORDER_FREQUENCY)

	for {
		select {
		case <-requestNetworkOrders:
			networkorders <- NetworkOrders
		case orderId := <-writeNetworkOrders:
			prev := NetworkOrders[orderId.Floor][orderId.Button]
			NetworkOrders[orderId.Floor][orderId.Button] = orderId.ID
			if prev != orderId.ID {
				SendNetworkOrders(NetworkOrders)
				elevator.UpdateLightsFromNetworkOrders(NetworkOrders)
				ticker.Reset(NETWORK_ORDER_FREQUENCY)
				if orderId.ID == "" {
					orderTimestamps[orderId.Floor][orderId.Button] = time.Time{}
				} else {
					orderTimestamps[orderId.Floor][orderId.Button] = time.Now()
				}
			}

		case <-ticker.C:
			// Send out NetworkOrders every time the ticker fires
			now := time.Now()
			for floor, floorOrders := range NetworkOrders {
				for button := range floorOrders {
					if NetworkOrders[floor][button] != "" && now.Sub(orderTimestamps[floor][button]) > config.ORDER_DEADLINE {
						assignOrder <- Orderstatus{Floor: floor, Button: ButtonType(button), Served: false}
						fmt.Println("Order expired, reassigning order: ", floor, button)
						orderTimestamps[floor][button] = now
					}
				}
			}

			SendNetworkOrders(NetworkOrders)
			elevator.UpdateLightsFromNetworkOrders(NetworkOrders)

		}
	}
}

func checkForUnavailable(
	requestSystemState chan<- bool,
	systemState <-chan map[string]Elev,
	nodeUnavailabe chan<- string) {

	unavailableIDs := make(map[string]bool)

	for {
		requestSystemState <- true
		localSystemState := <-systemState

		for id, elev := range localSystemState {
			if elev.State == EB_UNAVAILABLE {
				if _, alreadyUnavailable := unavailableIDs[id]; !alreadyUnavailable {
					unavailableIDs[id] = true
					fmt.Println("Elevator", id, "is unavailable")
					nodeUnavailabe <- id
				}
			} else {
				delete(unavailableIDs, id)
			}
		}
		time.Sleep(3 * time.Second)
	}
}

func CheckMissingConnToOrders(
	networkOrders [config.N_FLOORS][config.N_BUTTONS]string,
	nodeUnavailabe chan<- string) {

	processedIDs := make(map[string]bool)

	for _, floorButtons := range networkOrders {
		for _, id := range floorButtons {
			if id != "" && wranglerConnections[id] == nil && id != SELF_ID && !processedIDs[id] {
				nodeUnavailabe <- id
				processedIDs[id] = true
			}
		}
	}
}
