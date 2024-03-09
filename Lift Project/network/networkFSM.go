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

type World struct {
	Map map[string]Elev
}

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

func NetworkFSM(channels Channels, world *World) {
	networkChannels := NetworkChannels{
		DeputyPromotion:   make(chan map[string]Orderstatus),
		WranglerPromotion: make(chan bool),
		SheriffDead:       make(chan NetworkOrdersData),
		RelievedOfDuty:    make(chan bool),
		RemaindingOrders:  make(chan [config.N_FLOORS][config.N_BUTTONS]string),
	}

	state = st_initial

	for {
		switch state {
		case st_initial:
			sIP := wrangler.GetSheriffIP()
			if sIP == "" {

				NetworkOrders := [config.N_FLOORS][config.N_BUTTONS]string{}
				InitSherrif(channels, world, &NetworkOrders, "", networkChannels.RelievedOfDuty, networkChannels.RemaindingOrders)
				state = st_sherriff
			} else {
				fmt.Println("I am not the only Wrangler in town, connecting to Sheriff:")
				if wrangler.ConnectWranglerToSheriff(sIP) {
					fmt.Println("Me, a Wrangler connected to Sheriff")
					go wrangler.ReceiveMessageFromSheriff(channels.OrderAssigned, networkChannels)
					state = st_wrangler
				}
			}
			go orderForwarder(channels)
		case st_sherriff:
			//im jamming

			//check for sheriff Conflict
			sIP := wrangler.GetSheriffIP()
			//compare to own IP
			selfIP := strings.Split(string(config.Self_id), ":")
			//check for conflict
			if sIP != selfIP[0] {
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
					go wrangler.ReceiveMessageFromSheriff(channels.OrderAssigned, networkChannels)
					networkChannels.RelievedOfDuty <- true
					remaindingOrders := <-networkChannels.RemaindingOrders

					for i := 0; i < len(remaindingOrders); i++ {
						for j := 0; j < len(remaindingOrders[i]); j++ {
							if remaindingOrders[i][j] != "" {
								channels.OrderRequest <- Order{Floor: i, Button: elevio.ButtonType(j)}
							}
						}
					}

				}
			}

			// 	//the highest IP wins

		case st_wrangler:
			networkOrderData := <-networkChannels.SheriffDead

			if networkOrderData.TheChosenOne {
				fmt.Println("I am the chosen one, I am the Sheriff!")
				InitSherrif(channels, world, &networkOrderData.NetworkOrders, sheriffID, networkChannels.RelievedOfDuty, networkChannels.RemaindingOrders)
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
	channels Channels,
	world *World,
	networkorders *[config.N_FLOORS][config.N_BUTTONS]string,
	oldSheriff string, relievedOfDuty <-chan bool,
	remaindingOrders chan<- [config.N_FLOORS][config.N_BUTTONS]string) {

	nodeLeftNetwork := make(chan string)
	quitAssigner := make(chan bool)
	fmt.Println("I am the only Wrangler in town, I am the Sheriff!")
	NetworkUpdate := make(chan bool)
	go sheriff.Sheriff(channels.IncomingOrder, networkorders, nodeLeftNetwork, NetworkUpdate, relievedOfDuty, quitAssigner)
	go Assigner(channels, world, networkorders, NetworkUpdate, nodeLeftNetwork, channels.IncomingOrder, quitAssigner, remaindingOrders)
	//go redistributor(nodeLeftNetwork, channels.IncomingOrder, world, networkorders)
	if oldSheriff != "" {
		nodeLeftNetwork <- oldSheriff
		fmt.Println("Sending old sheriff to redistributer", oldSheriff)
	}
}
