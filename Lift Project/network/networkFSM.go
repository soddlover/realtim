package network

import (
	. "Project/config"
	"Project/elevator"
	"Project/network/SheriffWrangler/sheriff"
	"Project/network/SheriffWrangler/wrangler"
	"Project/network/systemStateSynchronizer"
	. "Project/types"
	"fmt"
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
	var startUDPListenerOnce sync.Once
	var deputy string = SELF_ID
	var latestNetworkOrderData NetworkOrderPacket

	requestSystemState := make(chan bool, NETWORK_BUFFER_SIZE)
	systemState := make(chan map[string]Elev, NETWORK_BUFFER_SIZE)
	nodeLeftNetwork := make(chan string, NETWORK_BUFFER_SIZE)
	assignOrder := make(chan Orderstatus, NETWORK_BUFFER_SIZE)
	recievedNetworkOrders := make(chan NetworkOrderPacket, NETWORK_BUFFER_SIZE)
	sheriffDead := make(chan bool)
	sheriffIP := make(chan string)

	go systemStateSynchronizer.SystemStateSynchronizer(
		requestSystemState,
		nodeLeftNetwork,
		elevatorStateBcast,
		systemState)

	go closeTCPConnections(nodeLeftNetwork, sheriffIP)

	currentDuty = dt_initial
	for {
		switch currentDuty {

		case dt_initial:

			sIP := wrangler.GetSheriffIP()
			if sIP == "" {
				fmt.Println("Someone shot the sherrif, but they didn't shoot the deputy...", deputy, SELF_ID)
				if deputy == SELF_ID {
					fmt.Println("As the former deputy!! I am promoted to Sheriff!")
					currentDuty = dt_sherriff

					go sheriff.Sheriff(assignOrder,
						latestNetworkOrderData,
						addToLocalQueue,
						requestSystemState,
						systemState)

				} else {
					time.Sleep(time.Second)
					deputy = SELF_ID
					continue
				}
			} else {
				fmt.Println("Attempting Connecting to Sheriff...")
				if wrangler.ConnectWranglerToSheriff(sIP) {
					sheriffIP <- sIP
					go wrangler.ReceiveTCPMessageFromSheriff(sheriffDead, addToLocalQueue)
					startUDPListenerOnce.Do(func() {
						go wrangler.ReceiveUDPNodeOrders(recievedNetworkOrders)
					})
					currentDuty = dt_wrangler
					fmt.Println("Suceessfully connected to Sheriff, assuming wrangler duties!")
				}
			}
			startOrderForwarderOnce.Do(func() {
				go orderForwarder(assignOrder, addToLocalQueue, localOrderRequest, localOrderServed)
			})

		case dt_sherriff:
			selfIP := strings.Split(string(SELF_ID), ":")[0]
			sIP := wrangler.GetSheriffIP()
			if sIP == "" {
				fmt.Println("Diconnect from network detected")
				sheriff.CloseAllConnections()
				currentDuty = dt_offline
			} else if sIP != selfIP {
				fmt.Println("Challenger detected, commencing shootout...")
				if sIP > selfIP {
					fmt.Println("I lost the shootout, goodbye cruel world!")
					os.Exit(1)
				} else {
					fmt.Println("This town ain't big enough for the two of us. I'm the only sheriff around here!")
					time.Sleep(5 * time.Second)
				}
			}
			time.Sleep(time.Second)

		case dt_wrangler:

			select {
			case <-sheriffDead:
				fmt.Println("Sheriff disconnect detected", latestNetworkOrderData)
				deputy = latestNetworkOrderData.DeputyID
				currentDuty = dt_initial
			case latestNetworkOrderData = <-recievedNetworkOrders:
				wrangler.CheckSync(requestSystemState, systemState, latestNetworkOrderData.Orders, addToLocalQueue)
				elevator.UpdateLightsFromNetworkOrders(latestNetworkOrderData.Orders)
			}

		case dt_offline:

			sIP := wrangler.GetSheriffIP()
			if sIP != "" {
				fmt.Println("Reconnected to network. Restarting...")
				os.Exit(1)
			}
			time.Sleep(time.Second)
		}
	}
}

func closeTCPConnections(
	lostConns <-chan string,
	sheriffID <-chan string) {

	var lastSheriffID string

	for {
		select {
		case id := <-lostConns:

			if id == SELF_ID {
				continue
			}
			fmt.Println("Lost connection to:", id)
			if currentDuty == dt_sherriff {
				sheriff.CloseConns(id)
			}
			id = strings.Split(id, ":")[0]
			if currentDuty == dt_wrangler && lastSheriffID == id {
				wrangler.CloseSheriffConn()
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
	localOrderServed <-chan Orderstatus) {

	for {
		select {

		case order := <-localOrderRequest:

			orderstat := Orderstatus{Floor: order.Floor, Button: order.Button, Served: false}

			if order.Button == BT_Cab {
				addToLocalQueue <- Order{Floor: order.Floor, Button: order.Button}
				continue
			}

			if currentDuty != dt_offline {
				if currentDuty == dt_sherriff {
					assignOrder <- orderstat
				} else {
					wrangler.SendOrderToSheriff(orderstat)
				}
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
