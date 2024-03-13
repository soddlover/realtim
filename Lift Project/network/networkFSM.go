package network

import (
	"fmt"
	"mymodule/config"
	"mymodule/elevator/elevio"
	"mymodule/network/SheriffDeputyWrangler/sheriff"
	"mymodule/network/SheriffDeputyWrangler/wrangler"
	"mymodule/systemStateSynchronizer"
	. "mymodule/types"
	"os"
	"time"
)

type duty int

const (
	dt_initial duty = iota
	dt_sherriff
	dt_deputy
	dt_wrangler
	dt_offline
)

var currentDuty duty

func NetworkFSM(
	elevatorState <-chan Elev,
	orderRequest chan Order,
	orderAssigned chan Order,
	orderDelete chan Orderstatus,
	incommingOrder chan Orderstatus,

) {

	requestSystemState := make(chan bool, 40)
	systemState := make(chan map[string]Elev, 40)
	nodeLeftNetwork := make(chan string, 40)

	go systemStateSynchronizer.SystemStateSynchronizer(
		requestSystemState,
		nodeLeftNetwork,
		elevatorState,
		systemState,
	)

	networkOrders := &NetworkOrders{}

	sheriffDead := make(chan NetworkOrdersData)
	relievedOfDuty := make(chan bool)
	remainingOrders := make(chan [config.N_FLOORS][config.N_BUTTONS]string)
	sheriffIP := make(chan string)

	//lostConns := make(chan string)
	go CloseTCPConns(nodeLeftNetwork, sheriffIP)
	//go Heartbeats(lostConns, systemState)
	go checkSync(requestSystemState, systemState, networkOrders, orderAssigned)

	currentDuty = dt_initial
	for {
		switch currentDuty {
		case dt_initial:
			sIP := wrangler.GetSheriffIP()
			if sIP == "" {

				InitSherrif(
					incommingOrder,
					requestSystemState,
					systemState,
					networkOrders,
					relievedOfDuty,
					remainingOrders,
					orderAssigned)
				currentDuty = dt_sherriff
			} else {
				fmt.Println("I am not the only Wrangler in town, connecting to Sheriff:")
				if wrangler.ConnectWranglerToSheriff(sIP) {
					fmt.Println("Me, a Wrangler connected to Sheriff")
					sheriffIP <- sIP
					go wrangler.ReceiveMessageFromSheriff(orderAssigned, sheriffDead, networkOrders)
					currentDuty = dt_wrangler
				}
			}
			go orderForwarder(incommingOrder, orderAssigned, orderRequest, orderDelete)
		case dt_sherriff:

			sIP := wrangler.GetSheriffIP()

			if sIP == "" {
				fmt.Println("This is weird, I should have been broadcasting my IP, read '' as broadcasted IP")
				fmt.Println("I must be offline so sad")
				currentDuty = dt_offline
				//relievedOfDuty <- true
			}
			time.Sleep(1 * time.Second)

		case dt_wrangler:
			networkOrderData := <-sheriffDead

			if networkOrderData.TheChosenOne {
				fmt.Println("I am the chosen one, I am the Sheriff!")
				networkOrders.Mutex.Lock()
				networkOrders.Orders = networkOrderData.NetworkOrders
				networkOrders.Mutex.Unlock()
				InitSherrif(
					incommingOrder,
					requestSystemState,
					systemState,
					networkOrders,
					relievedOfDuty,
					remainingOrders,
					orderAssigned)
				currentDuty = dt_sherriff

			} else {
				fmt.Println("I am not the chosen one, I am a Deputy")
				currentDuty = dt_initial
			}

			//listen for incoming orders
			//listen for new peers
			//listen for lost peers
			//listen for orders to delete
			//listen for orders to assign
		case dt_offline:
			sIP := wrangler.GetSheriffIP()
			if sIP != "" {
				fmt.Println("Back online, time to restart")
				os.Exit(1)
			}
			time.Sleep(1 * time.Second)
			//listen for incoming orders
			//listen for new peers
			//listen for lost peers
			//listen for orders to delete
			//listen for orders to assign
		}
	}
}

func CloseTCPConns(lostConns <-chan string, sheriffID <-chan string) {
	var lastSheriffID string
	for {
		select {
		case id := <-lostConns:
			if id == config.Self_id {
				fmt.Println("I am the lost connection, I dont have a TCP connection to my self to close")
				continue
			}
			if currentDuty == dt_sherriff {
				sheriff.CloseConns(id)
			}

			if currentDuty == dt_wrangler && lastSheriffID == id {
				wrangler.CloseSheriffConn()
			}

		case id := <-sheriffID:
			lastSheriffID = id
		}
	}
}

func InitSherrif(
	incomingOrder chan Orderstatus,
	requestSystemState chan<- bool,
	systemState <-chan map[string]Elev,
	networkorders *NetworkOrders,
	relievedOfDuty <-chan bool,
	remainingOrders chan<- [config.N_FLOORS][config.N_BUTTONS]string,
	orderAssigned chan<- Order,
) {
	nodeLeftNetwork := make(chan string)
	quitAssigner := make(chan bool)
	fmt.Println("I am the only Wrangler in town, I am the Sheriff!")
	networkUpdate := make(chan bool)
	go sheriff.Sheriff(incomingOrder, networkorders, nodeLeftNetwork, networkUpdate, relievedOfDuty, quitAssigner)
	go Assigner(
		networkUpdate,
		orderAssigned,
		requestSystemState,
		systemState,
		networkorders,
		nodeLeftNetwork,
		incomingOrder,
		quitAssigner,
		remainingOrders)
	go redistributor(
		nodeLeftNetwork,
		incomingOrder,
		requestSystemState,
		systemState,
		networkorders)
}

func orderForwarder(
	incomingOrder chan<- Orderstatus,
	orderAssigned chan<- Order,
	orderRequest <-chan Order,
	orderDelete <-chan Orderstatus,
) {
	for {
		select {
		case order := <-orderRequest:
			orderstat := Orderstatus{Owner: config.Self_id, Floor: order.Floor, Button: order.Button, Served: false}
			if order.Button == elevio.BT_Cab {
				orderAssigned <- Order{Floor: order.Floor, Button: order.Button}
				continue
			}
			if currentDuty == dt_offline {
				continue
			}

			if currentDuty == dt_sherriff {
				incomingOrder <- orderstat
			} else {
				wrangler.SendOrderToSheriff(orderstat)
			}
		case orderstat := <-orderDelete:
			if currentDuty == dt_sherriff || currentDuty == dt_offline {
				incomingOrder <- orderstat
			} else {
				wrangler.SendOrderToSheriff(orderstat)
			}
		}
	}
}

func checkSync(requestSystemState chan<- bool, systemState <-chan map[string]Elev, networkOrders *NetworkOrders, orderAssigned chan<- Order) {
	for {
		networkOrders.Mutex.Lock()
		for floor := 0; floor < config.N_FLOORS; floor++ {
			for button := 0; button < config.N_BUTTONS; button++ {
				if networkOrders.Orders[floor][button] != "" {
					requestSystemState <- true
					localSystemState := <-systemState
					assignedElev, existsInSystemState := localSystemState[networkOrders.Orders[floor][button]]
					if !existsInSystemState || !assignedElev.Queue[floor][button] {
						if networkOrders.Orders[floor][button] == config.Self_id {
							//delete(systemState, networkOrders[floor][button])
							//networkOrders[floor][button] = ""

							//elev := systemState[config.Self_id]
							// Modify the struct
							//elev.Queue[floor][button] = true
							// Put the struct back into the map
							//systemState[config.Self_id] = elev
							networkOrders.Mutex.Unlock()
							orderAssigned <- Order{Floor: floor, Button: elevio.ButtonType(button)}
							networkOrders.Mutex.Lock()
							fmt.Println("WARNING - Order not in sync with system state, reassigning order TO MYSELF KJØH")
						}
					}
				}
			}
		}
		networkOrders.Mutex.Unlock()
		time.Sleep(5 * time.Second)
	}
}
