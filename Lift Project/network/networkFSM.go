package network

import (
	"fmt"
	"mymodule/config"
	"mymodule/elevator/elevio"
	"mymodule/network/SheriffDeputyWrangler/sheriff"
	"mymodule/network/SheriffDeputyWrangler/wrangler"
	. "mymodule/types"
	"strings"
)

type duty int

const (
	dt_initial duty = iota
	dt_sherriff
	dt_deputy
	dt_wrangler
	dt_recovery
)

var currentDuty duty

func NetworkFSM(
	orderRequest chan Order,
	orderAssigned chan Orderstatus,
	orderDelete chan Orderstatus,
	systemState *SystemState,
	incommingOrder chan Orderstatus,
) {

	sheriffDead := make(chan NetworkOrdersData)
	relievedOfDuty := make(chan bool)
	remainingOrders := make(chan [config.N_FLOORS][config.N_BUTTONS]string)

	currentDuty = dt_initial

	for {
		switch currentDuty {
		case dt_initial:
			sIP := wrangler.GetSheriffIP()
			if sIP == "" {

				NetworkOrders := [config.N_FLOORS][config.N_BUTTONS]string{}
				InitSherrif(incommingOrder, systemState, &NetworkOrders, relievedOfDuty, remainingOrders, orderAssigned)
				currentDuty = dt_sherriff
			} else {
				fmt.Println("I am not the only Wrangler in town, connecting to Sheriff:")
				if wrangler.ConnectWranglerToSheriff(sIP) {
					fmt.Println("Me, a Wrangler connected to Sheriff")
					go wrangler.ReceiveMessageFromSheriff(orderAssigned, sheriffDead)
					currentDuty = dt_wrangler
				}
			}
			go orderForwarder(incommingOrder, orderAssigned, orderRequest, orderDelete)
		case dt_sherriff:
			//im jamming

			//check for sheriff Conflict
			sIP := wrangler.GetSheriffIP()
			//print("Sheriff IP: ", sIP, "\n")
			//compare to own IP
			selfIP := strings.Split(string(config.Self_id), ":")
			//check for conflict
			if sIP != "" && sIP != selfIP[0] {
				fmt.Println("Sheriff Conflict, my IP:", selfIP[0], "other Sheriff IP:", sIP)
				fmt.Println("Preparing for shootout!!!!")
				fmt.Println("Allahu Akbar")

				// 	//shootout
				if selfIP[0] > sIP {
					fmt.Println("I won the shootout! Theres a new sheriff in town.")
					continue
				}
				fmt.Println("I died.")

				if wrangler.ConnectWranglerToSheriff(sIP) {
					fmt.Println("Me, a Wrangler connected to Sheriff")
					currentDuty = dt_wrangler
					go wrangler.ReceiveMessageFromSheriff(orderAssigned, sheriffDead)
					relievedOfDuty <- true
					o := <-remainingOrders

					for i := 0; i < len(o); i++ {
						for j := 0; j < len(o[i]); j++ {
							if o[i][j] != "" {
								orderRequest <- Order{Floor: i, Button: elevio.ButtonType(j)}
							}
						}
					}
				}
			}

			// 	//the highest IP wins

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
		case dt_recovery:
			currentDuty = dt_initial
			//listen for incoming orders
			//listen for new peers
			//listen for lost peers
			//listen for orders to delete
			//listen for orders to assign
		}
	}
}

func InitSherrif(
	incomingOrder chan Orderstatus,
	systemState *SystemState,
	networkorders *[config.N_FLOORS][config.N_BUTTONS]string,
	relievedOfDuty <-chan bool,
	remainingOrders chan<- [config.N_FLOORS][config.N_BUTTONS]string,
	orderAssigned chan<- Orderstatus,
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
	orderAssigned chan<- Orderstatus,
	orderRequest <-chan Order,
	orderDelete <-chan Orderstatus,
) {
	for {
		select {
		case order := <-orderRequest:
			orderstat := Orderstatus{Owner: config.Self_id, Floor: order.Floor, Button: order.Button, Served: false}
			if order.Button == elevio.BT_Cab {
				orderAssigned <- orderstat
				continue
			}
			if currentDuty == dt_sherriff {
				incomingOrder <- orderstat
			} else {
				wrangler.SendOrderToSheriff(orderstat)
			}
		case orderstat := <-orderDelete:
			if currentDuty == dt_sherriff {
				incomingOrder <- orderstat
			} else {
				wrangler.SendOrderToSheriff(orderstat)
			}
		}
	}
}
