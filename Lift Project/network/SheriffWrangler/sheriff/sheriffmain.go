package sheriff

import (
	"fmt"
	"mymodule/config"
	"mymodule/elevator"
	"mymodule/elevator/elevio"
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
	networkorders := make(chan [config.N_FLOORS][config.N_BUTTONS]string)
	requestNetworkOrders := make(chan bool)

	ip := strings.Split(string(config.Id), ":")[0]
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
		assignOrder,
		requestSystemState,
		systemState,
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
	ticker := time.NewTicker(200 * time.Millisecond)

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
				ticker.Reset(200 * time.Millisecond)
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
						assignOrder <- Orderstatus{Floor: floor, Button: elevio.ButtonType(button), Served: false}
						fmt.Println("Order expired, reassigning order: ", floor, button)
						orderTimestamps[floor][button] = now
					}
				}
			}

			SendNetworkOrders(NetworkOrders)
			fmt.Println("Sending out NetworkOrders")
			elevator.UpdateLightsFromNetworkOrders(NetworkOrders)

		}
	}
}

func checkForUnavailable(
	requestSystemState chan<- bool,
	systemState <-chan map[string]Elev,
	nodeUnavailabe chan<- string,
) {
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

	for _, floorButtons := range networkOrders { //floor
		for _, id := range floorButtons {
			// The variable 'floor' represents the floor number. It's not used in this loop.

			//fmt.Printf("Checking order at floor %d, button %d, id: %s\n", floor, button, id) // Print the current order being checked
			if id != "" && wranglerConnections[id] == nil && id != config.Id && !processedIDs[id] {
				nodeUnavailabe <- id
				fmt.Println("***Missing connection to ACTIVE ORDER Reassigning order!!!***", id)
				processedIDs[id] = true
			}
		}
	}
}
