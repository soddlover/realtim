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

type State int

const (
	st_initial State = iota
	st_sherriff
	st_deputy
	st_wrangler
	st_recovery
)

var OnlineElevators = make(map[string]bool)
var state State
var sheriffID string

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

	state = st_initial

	for {
		switch state {
		case st_initial:
			sIP := wrangler.GetSheriffIP()
			if sIP == "" {

				NetworkOrders := [config.N_FLOORS][config.N_BUTTONS]string{}
				InitSherrif(incommingOrder, systemState, &NetworkOrders, "", relievedOfDuty, remainingOrders, orderAssigned)
				state = st_sherriff
			} else {
				fmt.Println("I am not the only Wrangler in town, connecting to Sheriff:")
				if wrangler.ConnectWranglerToSheriff(sIP) {
					fmt.Println("Me, a Wrangler connected to Sheriff")
					go wrangler.ReceiveMessageFromSheriff(orderAssigned, sheriffDead)
					state = st_wrangler
				}
			}
			go orderForwarder(incommingOrder, orderAssigned, orderRequest, orderDelete)
		case st_sherriff:
			//im jamming

			//check for sheriff Conflict
			sIP := wrangler.GetSheriffIP()
			print("Sheriff IP: ", sIP, "\n")
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
					state = st_wrangler
					go wrangler.ReceiveMessageFromSheriff(orderAssigned, sheriffDead)
					relievedOfDuty <- true
					remaindingOrders := <-remainingOrders

					for i := 0; i < len(remaindingOrders); i++ {
						for j := 0; j < len(remaindingOrders[i]); j++ {
							if remaindingOrders[i][j] != "" {
								orderRequest <- Order{Floor: i, Button: elevio.ButtonType(j)}
							}
						}
					}
				}
			}

			// 	//the highest IP wins

		case st_wrangler:
			networkOrderData := <-sheriffDead

			if networkOrderData.TheChosenOne {
				fmt.Println("I am the chosen one, I am the Sheriff!")
				InitSherrif(incommingOrder, systemState, &networkOrderData.NetworkOrders, sheriffID, relievedOfDuty, remainingOrders, orderAssigned)
				state = st_sherriff

			} else {
				fmt.Println("I am not the chosen one, I am a Deputy")
				state = st_initial
			}

			//listen for incoming orders
			//listen for new peers
			//listen for lost peers
			//listen for orders to delete
			//listen for orders to assign
		case st_recovery:
			state = st_initial
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
	oldSheriff string,
	relievedOfDuty <-chan bool,
	remaindingOrders chan<- [config.N_FLOORS][config.N_BUTTONS]string,
	orderAssigned chan<- Orderstatus,
) {

	nodeLeftNetwork := make(chan string)
	quitAssigner := make(chan bool)
	fmt.Println("I am the only Wrangler in town, I am the Sheriff!")
	networkUpdate := make(chan bool)
	go sheriff.Sheriff(incomingOrder, networkorders, nodeLeftNetwork, networkUpdate, relievedOfDuty, quitAssigner)
	go Assigner(networkUpdate, orderAssigned, systemState, networkorders, nodeLeftNetwork, incomingOrder, quitAssigner, remaindingOrders)
	//go redistributor(nodeLeftNetwork, channels.IncomingOrder, world, networkorders)
	if oldSheriff != "" {
		nodeLeftNetwork <- oldSheriff
		fmt.Println("Sending old sheriff to redistributer", oldSheriff)
	}
}
