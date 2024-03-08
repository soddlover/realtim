package network

import (
	"fmt"
	"mymodule/config"
	. "mymodule/elevator"
	"mymodule/elevator/elevio"
	"mymodule/network/SheriffDeputyWrangler/sheriff"
	"mymodule/network/SheriffDeputyWrangler/wrangler"
	"mymodule/network/peers"
	. "mymodule/types"
	"time"
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

func PeerConnector(id string, world *World, channels Channels) {

	peerUpdateCh := make(chan peers.PeerUpdate)
	peerTxEnable := make(chan bool)

	println("PeerConnector started, transmitting id: ", id)
	go peers.Transmitter(config.Peer_port, id, peerTxEnable)
	go peers.Receiver(config.Peer_port, peerUpdateCh)
	go NetworkFSM(channels, world)
	//on startup wait for connections then check if only one is online

	//OutgoingOrder := make(chan Order)

	//This code is just to higlight which channels are available

}

func InitSherrif(channels Channels, world *World, networkorders *[config.N_FLOORS][config.N_BUTTONS]string, oldSheriff string) {
	nodeLeftNetwork := make(chan string)
	fmt.Println("I am the only Wrangler in town, I am the Sheriff!")
	NetworkUpdate := make(chan bool)
	go sheriff.Sheriff(channels.IncomingOrder, networkorders, nodeLeftNetwork, NetworkUpdate)
	go Assigner(channels, world, networkorders, NetworkUpdate)
	go redistributer(nodeLeftNetwork, channels.IncomingOrder, world, networkorders)
	if oldSheriff != "" {
		nodeLeftNetwork <- oldSheriff
		fmt.Println("Sending old sheriff to redistributer", oldSheriff)
	}

}

func redistributer(nodeLeftNetwork chan string, incomingOrder chan Orderstatus, world *World, NetworkOrders *[config.N_FLOORS][config.N_BUTTONS]string) {
	for {
		select {
		case peerid := <-nodeLeftNetwork:
			delete(world.Map, peerid)
			fmt.Printf("world.Map: %v\n", world.Map)
			fmt.Println("Node left network, redistributing orders")
			fmt.Println("NetworkOrders: ", NetworkOrders)
			//check for orders owned by the leaving node
			for floor := 0; floor < len(NetworkOrders); floor++ {
				for button := 0; button < len(NetworkOrders[button]); button++ {
					if NetworkOrders[floor][button] == peerid {
						// Send to assigner for reassignment
						incomingOrder <- Orderstatus{Floor: floor, Button: elevio.ButtonType(button), Status: false, Owner: peerid}
					}
				}
			}
		}
	}

}

func orderForwarder(channels Channels) {
	for {
		select {
		case order := <-channels.OrderRequest:
			orderstat := Orderstatus{Owner: config.Self_id, Floor: order.Floor, Button: order.Button, Status: false}
			if order.Button == elevio.BT_Cab {
				channels.OrderAssigned <- orderstat
				continue
			}
			if state == st_sherriff {
				channels.IncomingOrder <- orderstat
			} else {
				wrangler.SendOrderToSheriff(orderstat)
			}
		case orderstat := <-channels.OrderDelete:
			if state == st_sherriff {
				channels.IncomingOrder <- orderstat
			} else {
				wrangler.SendOrderToSheriff(orderstat)
			}
		}
	}
}

func Assigner(channels Channels, world *World, NetworkOrders *[config.N_FLOORS][config.N_BUTTONS]string, NetworkUpdate chan bool) {

	for {
		select {
		case order := <-channels.IncomingOrder:
			//channels.OrderAssigned <- order
			if order.Status {
				fmt.Println("Order being deletetet")
				NetworkOrders[order.Floor][order.Button] = ""
				NetworkUpdate <- true
				continue
			}

			best_id := config.Self_id
			best_duration := 1000000 * time.Second
			for id, elevator := range world.Map {
				if elevator.Obstr {
					fmt.Println("Elevator with id: ", id, " is obstructed")
				}
				if elevator.State == Undefined || elevator.Obstr {
					continue
				}

				duration := timeToServeRequest(elevator, order.Button, order.Floor)
				if duration < best_duration {
					best_duration = duration
					best_id = id
				}
			}
			assigned := NetworkOrders[order.Floor][order.Button]
			if assigned != "" {
				if elev, ok := world.Map[assigned]; ok {
					if !elev.Obstr && !(elev.State == Undefined) {
						//do nothing as its already assigned to a working elevator, could send an additional message to it incase?
						fmt.Println("Order already assigned to a working elevator")
						fmt.Println("SOOME PROBLEMS OCCUR HERE MAYBE???")
						best_id = assigned
					}
				}
			}
			order.Owner = best_id
			NetworkOrders[order.Floor][order.Button] = best_id
			NetworkUpdate <- true
			fmt.Println("Added to NetworkOrders maps")
			if best_id == config.Self_id {
				channels.OrderAssigned <- order
			} else {
				go sheriff.SendOrderMessage(best_id, order)
			}

		}
	}
}

func timeToServeRequest(e_old Elev, b elevio.ButtonType, f int) time.Duration { //FIX THIS FUCKING FUNCTION
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("*****************************************")
			fmt.Println("Recovered from panic in timeToServeRequest:", r)
			fmt.Println("*****************************************")
			fmt.Println(e_old)
		}
	}()

	e := e_old
	e.Queue[f][b] = true

	arrivedAtRequest := 0

	ifEqual := func(inner_b elevio.ButtonType, inner_f int) {
		if inner_b == b && inner_f == f {
			arrivedAtRequest = 1
		}
	}

	duration := 0 * time.Second

	switch e.State {
	case EB_Idle:
		e.Dir = ChooseDirection(e)
		if e.Dir == DirStop {
			return duration
		}
	case EB_Moving:
		duration += config.TRAVEL_TIME / 2
		e.Floor += int(e.Dir)
	case EB_DoorOpen:
		duration -= config.DOOR_OPEN_TIME / 2
		if !OrdersAbove(e) && !OrdersBelow(e) {
			return duration
		}
	}

	for {
		if ShouldStop(e) {
			e = requestsClearAtCurrentFloor(e, ifEqual)
			if arrivedAtRequest == 1 {
				return duration
			}
			duration += config.DOOR_OPEN_TIME
			e.Dir = ChooseDirection(e)
		}
		e.Floor += int(e.Dir)
		duration += config.TRAVEL_TIME
	}
}

func requestsClearAtCurrentFloor(e_old Elev, f func(elevio.ButtonType, int)) Elev {
	e := e_old
	for b := elevio.ButtonType(0); b < config.N_BUTTONS; b++ {
		if e.Queue[e.Floor][b] {
			e.Queue[e.Floor][b] = false
			if f != nil {
				f(b, e.Floor)
			}
		}
	}
	return e
}
