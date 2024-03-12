package network

import (
	"fmt"
	"mymodule/config"
	"mymodule/elevator/elevio"
	"mymodule/network/SheriffDeputyWrangler/sheriff"
	"mymodule/network/SheriffDeputyWrangler/wrangler"
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
	orderRequest chan Order,
	orderAssigned chan Order,
	orderDelete chan Orderstatus,
	systemState map[string]Elev,
	incommingOrder chan Orderstatus,
) {

	sheriffDead := make(chan NetworkOrdersData)
	relievedOfDuty := make(chan bool)
	remainingOrders := make(chan [config.N_FLOORS][config.N_BUTTONS]string)
	NetworkOrders := [config.N_FLOORS][config.N_BUTTONS]string{}
	currentDuty = dt_initial
	lostConns := make(chan string)
	go CheckHeartbeats(lostConns)
	go Heartbeats(lostConns)
	go checkSync(systemState, &NetworkOrders, orderAssigned)

	for {
		switch currentDuty {
		case dt_initial:
			sIP := wrangler.GetSheriffIP()
			if sIP == "" {

				InitSherrif(incommingOrder, systemState, &NetworkOrders, relievedOfDuty, remainingOrders, orderAssigned)
				currentDuty = dt_sherriff
			} else {
				fmt.Println("I am not the only Wrangler in town, connecting to Sheriff:")
				if wrangler.ConnectWranglerToSheriff(sIP) {
					fmt.Println("Me, a Wrangler connected to Sheriff")
					go wrangler.ReceiveMessageFromSheriff(orderAssigned, sheriffDead, &NetworkOrders)
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

			/*
				case sIP == "offline":
					fmt.Println("Once i was afraid i was petrified, killing myself")
					os.Exit(1)

				case sIP == "DISCONNECTED":
					fmt.Println("Something is wrong, read ", sIP, " as broadcasted IP")

				case sIP != selfIP[0]:
					fmt.Println("Sheriff Conflict, my IP:", selfIP[0], "other Sheriff IP:", sIP)
					fmt.Println("Preparing for shootout!!!!")
					fmt.Println("Allahu Akbar")

					if selfIP[0] > sIP {
						fmt.Println("I won the shootout! Theres a new sheriff in town.")
						time.Sleep(1 * time.Second)
						continue
					}
					fmt.Println("I died.")

					if wrangler.ConnectWranglerToSheriff(sIP) {
						fmt.Println("Me, a Wrangler connected to Sheriff")
						currentDuty = dt_wrangler
						go wrangler.ReceiveMessageFromSheriff(orderAssigned, sheriffDead, &NetworkOrders)
						relievedOfDuty <- true
						o := <-remainingOrders
						time.Sleep(5 * time.Second)

						for i := 0; i < len(o); i++ {
							for j := 0; j < len(o[i]); j++ {
								if o[i][j] != "" {
									orderRequest <- Order{Floor: i, Button: elevio.ButtonType(j)}
								}
							}
						}
						fmt.Println("Transferred orders to new Sheriff")
					}
			*/

		case dt_wrangler:
			networkOrderData := <-sheriffDead

			if networkOrderData.TheChosenOne {
				fmt.Println("I am the chosen one, I am the Sheriff!")
				InitSherrif(incommingOrder, systemState, &networkOrderData.NetworkOrders, relievedOfDuty, remainingOrders, orderAssigned)
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

func Heartbeats(lostConns <-chan string) {
	for {
		id := <-lostConns
		if currentDuty == dt_sherriff {
			sheriff.CloseConns(id)
		} else {
			wrangler.CloseSheriffConn()
		}
	}
}

func InitSherrif(
	incomingOrder chan Orderstatus,
	systemState map[string]Elev,
	networkorders *[config.N_FLOORS][config.N_BUTTONS]string,
	relievedOfDuty <-chan bool,
	remainingOrders chan<- [config.N_FLOORS][config.N_BUTTONS]string,
	orderAssigned chan<- Order,
) {

	nodeLeftNetwork := make(chan string)
	quitAssigner := make(chan bool)
	fmt.Println("I am the only Wrangler in town, I am the Sheriff!")
	networkUpdate := make(chan bool)
	go sheriff.Sheriff(incomingOrder, networkorders, nodeLeftNetwork, networkUpdate, relievedOfDuty, quitAssigner)
	go Assigner(networkUpdate, orderAssigned, systemState, networkorders, nodeLeftNetwork, incomingOrder, quitAssigner, remainingOrders)
	//go redistributor(nodeLeftNetwork, channels.IncomingOrder, world, networkorders)
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

func checkSync(systemState map[string]Elev, networkOrders *[config.N_FLOORS][config.N_BUTTONS]string, orderAssigned chan<- Order) {
	for {
		for floor := 0; floor < config.N_FLOORS; floor++ {
			for button := 0; button < config.N_BUTTONS; button++ {
				if networkOrders[floor][button] != "" {
					_, existsInSystemState := systemState[networkOrders[floor][button]]
					if !existsInSystemState || !systemState[networkOrders[floor][button]].Queue[floor][button] {

						if networkOrders[floor][button] == config.Self_id {
							orderAssigned <- Order{Floor: floor, Button: elevio.ButtonType(button)}
							fmt.Println("WARNING - Order not in sync with system state, reassigning order TO MYSELF KJØH")
						}
					}
				}
			}
		}

		time.Sleep(1 * time.Second)
	}
}
