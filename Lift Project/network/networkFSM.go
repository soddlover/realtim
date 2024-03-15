package network

import (
	"fmt"
	"mymodule/config"
	"mymodule/elevator"
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
	dt_initial  duty = 0
	dt_sherriff duty = 1
	dt_wrangler duty = 2
	dt_offline  duty = 3
)

func NetworkFSM(
	elevatorStateBcast <-chan Elev,
	localOrderRequest <-chan Order,
	addToLocalQueue chan<- Order,
	localOrderServed <-chan Orderstatus,
) {

	var startOrderForwarderOnce sync.Once
	var lastnetworkOrders [config.N_FLOORS][config.N_BUTTONS]string
	var chosenOne bool = true
	var latestNetworkOrderData NetworkOrdersData

	requestSystemState := make(chan bool, config.NETWORK_BUFFER_SIZE)
	systemState := make(chan map[string]Elev, config.NETWORK_BUFFER_SIZE)
	nodeLeftNetwork := make(chan string, config.NETWORK_BUFFER_SIZE)
	assignOrder := make(chan Orderstatus, config.NETWORK_BUFFER_SIZE)
	recievedNetworkOrders := make(chan NetworkOrdersData, config.NETWORK_BUFFER_SIZE)

	go systemStateSynchronizer.SystemStateSynchronizer(
		requestSystemState,
		nodeLeftNetwork,
		elevatorStateBcast,
		systemState,
	)

	sheriffDead := make(chan bool)
	sheriffIP := make(chan string)
	go DisconnectHandeler(nodeLeftNetwork, sheriffIP, sheriffDead)

	currentDuty = dt_initial
	for {
		switch currentDuty {
		case dt_initial:
			sIP := wrangler.GetSheriffIP()
			if sIP == "" {
				if chosenOne {
					fmt.Println("I am sheriff!")
					currentDuty = dt_sherriff
					go sheriff.Sheriff(assignOrder,
						lastnetworkOrders,
						addToLocalQueue,
						requestSystemState,
						systemState)

				} else {
					time.Sleep(1 * time.Second)
					chosenOne = true
					continue
				}
			} else {
				fmt.Println("Attempting Connecting to Sheriff:")
				if wrangler.ConnectWranglerToSheriff(sIP) {
					sheriffIP <- sIP
					go wrangler.ReceiveTCPMessageFromSheriff(sheriffDead, recievedNetworkOrders, addToLocalQueue)
					go wrangler.ReceiveUDPNodeOrders(recievedNetworkOrders)
					currentDuty = dt_wrangler
					fmt.Println("I am wrangler!")
				}
			}
			startOrderForwarderOnce.Do(func() {
				go orderForwarder(assignOrder, addToLocalQueue, localOrderRequest, localOrderServed)
			})
		case dt_sherriff:

			sIP := wrangler.GetSheriffIP()

			if sIP == "" {
				fmt.Println("I have gone offline closing all connections")
				sheriff.CloseConns("ALL")
				currentDuty = dt_offline
				//relievedOfDuty <- true
			}
			time.Sleep(1 * time.Second)

		case dt_wrangler:
			select {
			case <-sheriffDead:
				lastnetworkOrders = latestNetworkOrderData.NetworkOrders
				chosenOne = latestNetworkOrderData.TheChosenOne
				currentDuty = dt_initial
			case latestNetworkOrderData := <-recievedNetworkOrders:
				wrangler.CheckSync(requestSystemState, systemState, latestNetworkOrderData.NetworkOrders, addToLocalQueue)
				elevator.UpdateLightsFromNetworkOrders(latestNetworkOrderData.NetworkOrders)
			}
		case dt_offline:

			sIP := wrangler.GetSheriffIP()
			if sIP != "" {
				fmt.Println("Coming back online, restarting...")
				os.Exit(1)
			}
			time.Sleep(1 * time.Second)
		}
	}
}

func DisconnectHandeler(lostConns <-chan string, sheriffID <-chan string, sheriffDead chan<- bool) {
	var lastSheriffID string
	for {
		select {
		case id := <-lostConns:

			if id == config.Id {
				continue
			}
			fmt.Println("Lost connection to:", id)
			if currentDuty == dt_sherriff {
				sheriff.CloseConns(id)
			}
			id = strings.Split(id, ":")[0]
			if currentDuty == dt_wrangler && lastSheriffID == id {
				wrangler.CloseSheriffConn()
				sheriffDead <- true

			}

		case id := <-sheriffID:
			lastSheriffID = id
		}
	}
}

func orderForwarder(
	assignOrder chan<- Orderstatus,
	addToLocalQueue chan<- Order,
	localOrderRequest <-chan Order,
	localOrderServed <-chan Orderstatus,
) {
	for {
		select {
		case order := <-localOrderRequest:
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
