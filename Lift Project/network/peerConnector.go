package network

import (
	"fmt"
	"mymodule/config"
	. "mymodule/elevator"
	"mymodule/elevator/elevio"
	"mymodule/network/SheriffDeputyWrangler/deputy"
	"mymodule/network/SheriffDeputyWrangler/sheriff"
	"mymodule/network/SheriffDeputyWrangler/wrangler"
	"mymodule/network/peers"
	. "mymodule/types"
	"time"

	"github.com/google/uuid"
)

type OrderAndID struct {
	Order Order
	ID    string
}

type World struct {
	Map map[string]Elev
}

type State int

const (
	initial State = iota
	sherriff
	deputy
	wrangler
	recovery
)

var OnlineElevators = make(map[string]bool)
var state State

func NetworkFSM(channels Channels, world *World) {
	state = initial
	for {
		switch state {
		case initial:
			sIP := sheriff.GetSheriffIP()
			if sIP == "" {
				NetworkOrders := make(map[string]Orderstatus)
				InitSherrif(channels, world, NetworkOrders)
				state = sherriff
			} else {
				fmt.Println("I am not the only Wrangler in town, connecting to Sheriff:")
				if sheriff.ConnectWranglerToSheriff(sIP) {
					fmt.Println("Me, a Wrangler connected to Sheriff")
					go sheriff.ReceiveMessageFromSheriff(channels.OrderAssigned)
					go orderForwarder(channels)
					state = wrangler
				}
			}
		case sherriff:
			//im jamming
		case deputy:
			select {
			case <-deputyPromotion:
				initSheriff()
				state = sherriff
			}

		case wrangler:
			select {
			case <-wranglerPromotion:
				initDeputy()
				state = deputy
			}

			//listen for incoming orders
			//listen for new peers
			//listen for lost peers
			//listen for orders to delete
			//listen for orders to assign
		case recovery:

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

	peerUpdateSheriff := make(chan peers.PeerUpdate)
	println("PeerConnector started, transmitting id: ", id)
	go peers.Transmitter(config.Peer_port, id, peerTxEnable)
	go peers.Receiver(config.Peer_port, peerUpdateCh)
	go peerUpdater(peerUpdateCh, world, peerUpdateSheriff)
	go NetworkFSM(channels, world)
	//on startup wait for connections then check if only one is online

	//OutgoingOrder := make(chan Order)

	//This code is just to higlight which channels are available
	select {
	case <-deputy.DeputyBecomeSheriff:
	case <-wrangler.WranglerPromotion:
	case <-wrangler.SheriffDisconnectedFromWrangler:
	}

}

func InitSherrif(channels Channels, world *World, NetworkOrders map[string]Orderstatus) {
	fmt.Println("I am the only Wrangler in town, I am the Sheriff!")
	nodeLeftNetwork := make(chan string)
	go sheriff.Sheriff(channels.IncomingOrder, NetworkOrders, nodeLeftNetwork)
	go orderForwarder(channels)
	go Assigner(channels, world, NetworkOrders)
	go redistributer(nodeLeftNetwork, channels.IncomingOrder, world, NetworkOrders)

}

func peerUpdater(peerUpdateCh chan peers.PeerUpdate, world *World, peerUpdateSheriff chan peers.PeerUpdate) {
	for {
		select {
		case p := <-peerUpdateCh:
			fmt.Printf("Peer update:\n")
			fmt.Printf("  Peers:    %q\n", p.Peers)
			fmt.Printf("  New:      %q\n", p.New)
			fmt.Printf("  Lost:     %q\n", p.Lost)
			// for _, peer := range p.Peers {
			// 	OnlineElevators[peer] = true
			// }
			// for _, element := range p.Lost {
			// 	//OnlineElevators[element] = false
			// 	//delete(world.Map, element)
			// 	//elevator := world.Map[element]
			// 	//elevator.State = Undefined
			// 	//world.Map[element] = elevator
			// 	//print("element was set as unavailable")
			// }

		}
	}
}

func redistributer(nodeLeftNetwork chan string, incomingOrder chan Orderstatus, world *World, NetworkOrders map[string]Orderstatus) {
	for {
		select {
		case peerid := <-nodeLeftNetwork:
			delete(world.Map, peerid)
			fmt.Println("Node left network, redistributing orders")
			//check for orders owned by the leaving node
			for _, order := range NetworkOrders {
				if order.Owner == peerid {
					//send to assigner for reassignment
					if order.Status {
						fmt.Println("Something is horribly wrong if you read this")
					}
					incomingOrder <- order
				}
			}
		}
	}

}

func orderForwarder(channels Channels) {
	for {
		select {
		case order := <-channels.OrderRequest:

			ID := uuid.New().String()
			orderstat := Orderstatus{OrderID: ID, Owner: config.Self_id, Floor: order.Floor, Button: order.Button, Status: false}
			if order.Button == elevio.BT_Cab {
				channels.OrderAssigned <- orderstat
				continue
			}
			if state == sherriff {
				channels.IncomingOrder <- orderstat
			} else {
				wrangler.SendOrderToSheriff(orderstat)
			}
		case orderstat := <-channels.OrderDelete:
			if state == sherriff {
				channels.IncomingOrder <- orderstat
			} else {
				wrangler.SendOrderToSheriff(orderstat)
			}

		}
	}

}

func Assigner(channels Channels, world *World, NetworkOrders map[string]Orderstatus) {

	for {
		select {
		case order := <-channels.IncomingOrder:
			//channels.OrderAssigned <- order
			if order.Status {
				fmt.Println("Order being deletetet")
				delete(NetworkOrders, order.OrderID)
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
			fmt.Println("Assigning order to elevator with id: ", best_id)
			order.Owner = best_id
			NetworkOrders[order.OrderID] = order
			fmt.Println("NetworkOrders: ", NetworkOrders)
			if best_id == config.Self_id {
				channels.OrderAssigned <- order
			} else {
				fmt.Println("Sending order to elevator with id: ", best_id)
				go sheriff.SendOrderMessage(best_id, order)
			}

		}
	}
}

func timeToServeRequest(e_old Elev, b elevio.ButtonType, f int) time.Duration { //FIX THIS FUCKING FUNCTION
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
