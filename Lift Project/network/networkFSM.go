package network

import (
	"fmt"
	"mymodule/config"
	"mymodule/network/SheriffDeputyWrangler/sheriff"
	"mymodule/network/SheriffDeputyWrangler/wrangler"
	. "mymodule/types"
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

var state State
var sheriffID string

func NetworkFSM(channels Channels, world *World) {
	networkChannels := NetworkChannels{
		DeputyPromotion:   make(chan map[string]Orderstatus),
		WranglerPromotion: make(chan bool),
		SheriffDead:       make(chan NetworkOrdersData),
		RelievedOfDuty:    make(chan bool),
	}
	state = st_initial

	for {
		switch state {
		case st_initial:
			sIP := wrangler.GetSheriffIP()
			if sIP == "" {

				NetworkOrders := [config.N_FLOORS][config.N_BUTTONS]string{}
				InitSherrif(channels, world, &NetworkOrders, "")
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
			//sheriff Conflict
			//Shootout
			//Fastest trigger in the west?

		case st_wrangler:
			networkOrderData := <-networkChannels.SheriffDead

			if networkOrderData.TheChosenOne {
				fmt.Println("I am the chosen one, I am the Sheriff!")
				InitSherrif(channels, world, &networkOrderData.NetworkOrders, sheriffID)
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

func InitSherrif(channels Channels, world *World, networkorders *[config.N_FLOORS][config.N_BUTTONS]string, oldSheriff string) {
	nodeLeftNetwork := make(chan string)
	fmt.Println("I am the only Wrangler in town, I am the Sheriff!")
	NetworkUpdate := make(chan bool)
	go sheriff.Sheriff(channels.IncomingOrder, networkorders, nodeLeftNetwork, NetworkUpdate)
	go Assigner(channels, world, networkorders, NetworkUpdate, nodeLeftNetwork, channels.IncomingOrder)
	//go redistributor(nodeLeftNetwork, channels.IncomingOrder, world, networkorders)
	if oldSheriff != "" {
		nodeLeftNetwork <- oldSheriff
		fmt.Println("Sending old sheriff to redistributer", oldSheriff)
	}
}
