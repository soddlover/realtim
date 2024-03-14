package network

import (
	"fmt"
	"mymodule/config"
	elevatorFSM "mymodule/elevator"
	"mymodule/elevator/elevio"
	"mymodule/network/SheriffWrangler/sheriff"
	"mymodule/network/SheriffWrangler/wrangler"
	"mymodule/systemStateSynchronizer"
	. "mymodule/types"
	"os"
	"strings"
	"sync"
	"time"
)

type duty int

var currentDuty duty

const (
	dt_initial duty = iota
	dt_sherriff
	dt_wrangler
	dt_offline
)
const N_FLOORS = config.N_FLOORS
const N_BUTTONS = config.N_BUTTONS

func NetworkFSM(
	elevatorStateBcast <-chan Elev,
	localRequest <-chan Order,
	addToLocalQueue chan<- Order,
	localOrderServed <-chan Orderstatus,
) {

	var startOrderForwarderOnce sync.Once
	var LastnetworkOrders [N_FLOORS][N_BUTTONS]string
	var ChosenOne bool = true

	requestSystemState := make(chan bool, 40)
	systemState := make(chan map[string]Elev, 40)
	nodeLeftNetwork := make(chan string, 40)
	assignOrder := make(chan Orderstatus, 10)

	go systemStateSynchronizer.SystemStateSynchronizer(
		requestSystemState,
		nodeLeftNetwork,
		elevatorStateBcast,
		systemState,
	)

	sheriffDead := make(chan NetworkOrdersData)
	sheriffIP := make(chan string)
	go CloseTCPConns(nodeLeftNetwork, sheriffIP)

	//go checkSync(requestSystemState, systemState, networkOrders, addToLocalQueue)

	currentDuty = dt_initial
	for {
		switch currentDuty {
		case dt_initial:
			fmt.Println("Currently in inital condition, attetming to get sheriff ip")
			sIP := wrangler.GetSheriffIP()
			fmt.Println("Sheriff IP is:", sIP)
			if sIP == "" {
				if ChosenOne {
					InitSherrif(
						assignOrder,
						requestSystemState,
						systemState,
						LastnetworkOrders,
						addToLocalQueue)
					currentDuty = dt_sherriff
				} else {
					fmt.Println("I am not the chosen one , attempting again")
					time.Sleep(1 * time.Second)
					ChosenOne = true
					continue
				}
			} else {
				fmt.Println("Attempting Connecting to Sheriff:")
				if wrangler.ConnectWranglerToSheriff(sIP) {
					fmt.Println("CONNECTED TO SHERIFF WAITING TO RECIEVE IOP")
					sheriffIP <- sIP
					go wrangler.ReceiveMessageFromSheriff(addToLocalQueue, sheriffDead)
					fmt.Println("IP recieved on channel, starting message reciever")
					currentDuty = dt_wrangler
				}
			}
			startOrderForwarderOnce.Do(func() {
				go orderForwarder(assignOrder, addToLocalQueue, localRequest, localOrderServed)
			})
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
			LastnetworkOrders = networkOrderData.NetworkOrders
			ChosenOne = networkOrderData.TheChosenOne
			fmt.Println("Lets double check if sherriff actually is dead")
			currentDuty = dt_initial

		case dt_offline:
			sIP := wrangler.GetSheriffIP()
			if sIP != "" {
				fmt.Println("Back online, time to restart")
				os.Exit(1)
			}
			time.Sleep(1 * time.Second)
		}
	}
}

func CloseTCPConns(lostConns <-chan string, sheriffID <-chan string) {
	var lastSheriffID string
	for {
		select {
		case id := <-lostConns:
			fmt.Println("Lost connection to:", id)
			fmt.Println("current state is:", currentDuty, "last sheriff", lastSheriffID)
			if id == config.Self_id {
				fmt.Println("I am the lost connection, I dont have a TCP connection to my self to close")
				continue
			}
			if currentDuty == dt_sherriff {
				fmt.Println("I am the Sheriff, I am closing the connection to:", id)
				sheriff.CloseConns(id)
			}
			id = strings.Split(id, ":")[0] //remove this??
			if currentDuty == dt_wrangler && lastSheriffID == id {
				fmt.Println("I am the Wrangler, I am closing the connection to:", id)
				wrangler.CloseSheriffConn()
			}

		case id := <-sheriffID:
			lastSheriffID = id
		}
	}
}

func InitSherrif(
	assignOrder chan Orderstatus,
	requestSystemState chan<- bool,
	systemState <-chan map[string]Elev,
	lastnetworkOrders [N_FLOORS][N_BUTTONS]string,
	addToLocalQueue chan<- Order,
) {

	nodeUnavailabe := make(chan string)
	requestNetworkOrders := make(chan bool)
	writeNetworkOrders := make(chan OrderID)
	networkorders := make(chan [config.N_FLOORS][config.N_BUTTONS]string)

	fmt.Println("I am the only Wrangler in town, I am the Sheriff!")

	go netWorkOrderHandler(
		requestNetworkOrders,
		writeNetworkOrders,
		networkorders,
		assignOrder,
		lastnetworkOrders)

	go sheriff.Sheriff(
		assignOrder,
		requestNetworkOrders,
		networkorders,
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

	go checkForUnavailable(
		requestSystemState,
		systemState,
		nodeUnavailabe)

}

func orderForwarder(
	assignOrder chan<- Orderstatus,
	addToLocalQueue chan<- Order,
	localRequest <-chan Order,
	localOrderServed <-chan Orderstatus,
) {
	for {
		select {
		case order := <-localRequest:
			orderstat := Orderstatus{Floor: order.Floor, Button: order.Button, Served: false}
			if order.Button == elevio.BT_Cab {
				addToLocalQueue <- Order{Floor: order.Floor, Button: order.Button}
				continue
			}
			if currentDuty == dt_offline {
				continue
			}

			if currentDuty == dt_sherriff {
				assignOrder <- orderstat
			} else {
				wrangler.SendOrderToSheriff(orderstat)
			}
		case orderstat := <-localOrderServed:
			if currentDuty == dt_sherriff || currentDuty == dt_offline {
				assignOrder <- orderstat
			} else {
				wrangler.SendOrderToSheriff(orderstat)
			}
		}
	}
}

func checkSync(requestSystemState chan<- bool, systemState <-chan map[string]Elev, networkOrders *NetworkOrders, addToLocalQueue chan<- Order) {
	for {
		networkOrders.Mutex.Lock()
		for floor := 0; floor < N_FLOORS; floor++ {
			for button := 0; button < N_BUTTONS; button++ {
				if networkOrders.Orders[floor][button] != "" {
					requestSystemState <- true
					localSystemState := <-systemState
					assignedElev, existsInSystemState := localSystemState[networkOrders.Orders[floor][button]]
					if !existsInSystemState || !assignedElev.Queue[floor][button] {
						if networkOrders.Orders[floor][button] == config.Self_id {
							addToLocalQueue <- Order{Floor: floor, Button: elevio.ButtonType(button)}
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

func netWorkOrderHandler(
	requestNetworkOrders <-chan bool,
	writeNetworkOrders <-chan OrderID,
	networkorders chan<- [N_FLOORS][N_BUTTONS]string,
	incomingOrder chan<- Orderstatus,
	lastNetworkOrders [N_FLOORS][N_BUTTONS]string) {

	NetworkOrders := lastNetworkOrders

	// Create a new ticker that fires every 3 seconds
	ticker := time.NewTicker(3 * time.Second)

	for {
		select {
		case <-requestNetworkOrders:
			networkorders <- NetworkOrders
		case orderId := <-writeNetworkOrders:
			NetworkOrders[orderId.Floor][orderId.Button] = orderId.ID
			sheriff.SendNetworkOrders(NetworkOrders)
			elevatorFSM.UpdateLightsFromNetworkOrders(NetworkOrders)
			ticker.Reset(3 * time.Second)
		case <-ticker.C:
			// Send out NetworkOrders every time the ticker fires
			sheriff.SendNetworkOrders(NetworkOrders)
			elevatorFSM.UpdateLightsFromNetworkOrders(NetworkOrders)
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
