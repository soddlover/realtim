package network

import (
	"fmt"
	"mymodule/config"
	. "mymodule/elevator"
	elevatorFMS "mymodule/elevator"
	elevatorFSM "mymodule/elevator"
	"mymodule/elevator/elevio"
	"mymodule/network/peers"
	"mymodule/network/supervisor"
	"time"

	"github.com/google/uuid"
)

type OrderAndID struct {
	Order Order
	ID    string
}

type World struct {
	Map map[string]elevatorFMS.Elev
}

var IsSuper bool = false
var OnlineElevators = make(map[string]bool)

func PeerConnector(id string, world *World, channels elevatorFMS.Channels) {

	peerUpdateCh := make(chan peers.PeerUpdate)
	peerTxEnable := make(chan bool)
	peerUpdateSupervisor := make(chan peers.PeerUpdate)
	println("PeerConnector started, transmitting id: ", id)
	go peers.Transmitter(config.Peer_port, id, peerTxEnable)
	go peers.Receiver(config.Peer_port, peerUpdateCh)
	go peerUpdater(peerUpdateCh, world, peerUpdateSupervisor)
	//on startup wait for connections then check if only one is online

	incomingOrder := make(chan supervisor.Orderstatus, 10)
	//OutgoingOrder := make(chan elevatorFMS.Order)

	sIP := supervisor.GetSupervisorIP()
	if sIP == "" {
		fmt.Println("I am the only one")
		go supervisor.Supervisor(incomingOrder)
		go orderForwarder(channels, incomingOrder)
		go Assigner(incomingOrder, channels.OrderAssigned, world)
		IsSuper = true

	} else {
		fmt.Println("I am not the only one connecting to SUPE")
		if supervisor.ConnectToSupervisor(sIP) {
			fmt.Println("Connected to supervisor")
			go supervisor.ReceiveOrderFromSupervisor(channels.OrderAssigned)
			go orderForwarder(channels, incomingOrder)
		}
	}

}

func peerUpdater(peerUpdateCh chan peers.PeerUpdate, world *World, peerUpdateSupervisor chan peers.PeerUpdate) {
	for {
		select {
		case p := <-peerUpdateCh:
			fmt.Printf("Peer update:\n")
			fmt.Printf("  Peers:    %q\n", p.Peers)
			fmt.Printf("  New:      %q\n", p.New)
			fmt.Printf("  Lost:     %q\n", p.Lost)
			for _, peer := range p.Peers {
				OnlineElevators[peer] = true
			}
			for _, element := range p.Lost {
				delete(world.Map, element)
				elevator := world.Map[element]
				elevator.State = elevatorFMS.Undefined
				world.Map[element] = elevator
				print("element was set as unavailable")
			}

		}
	}
}

func orderForwarder(channels Channels, incomingOrder chan supervisor.Orderstatus) {
	for {
		select {
		case order := <-channels.OrderRequest:
			if order.Button == elevio.BT_Cab {
				channels.OrderAssigned <- order
				continue
			}
			ID := uuid.New().String()
			orderstat := supervisor.Orderstatus{OrderID: ID, Owner: config.Self_id, Floor: order.Floor, Button: order.Button, Status: false}
			if IsSuper {
				incomingOrder <- orderstat
			} else {
				supervisor.SendOrderToSupervisor(orderstat)
			}
		}
	}

}

func Assigner(incomingOrder chan supervisor.Orderstatus, orderAssigned chan elevatorFSM.Order, world *World) {

	for {
		select {
		case order := <-incomingOrder:
			//channels.OrderAssigned <- order
			best_id := ""
			best_duration := 1000000
			for id, elevator := range world.Map {
				if elevator.State == Undefined {
					continue
				}
				duration := timeToServeRequestWithTimeout(elevator, order.Button, order.Floor)
				if duration < best_duration {
					best_duration = duration
					best_id = id
				}
			}
			fmt.Println("Assigning order to elevator with id: ", best_id)
			if best_id == config.Self_id {
				orderAssigned <- Order{Floor: order.Floor, Button: order.Button}
			} else {
				fmt.Println("Sending order to elevator with id: ", best_id)
				go supervisor.SendMessage(best_id, order)
			}

		}
	}
}
func timeToServeRequestWithTimeout(e_old Elev, b elevio.ButtonType, f int) int {
	resultChan := make(chan int)
	go func() {
		resultChan <- timeToServeRequest(e_old, b, f)
	}()

	select {
	case result := <-resultChan:
		return result
	case <-time.After(50 * time.Millisecond):
		fmt.Println("Timeout in timeToServeRequestWithTimeout")
		return 0
	}
}

func timeToServeRequest(e_old Elev, b elevio.ButtonType, f int) int { //FIX THIS FUCKING FUNCTION
	e := e_old
	e.Queue[f][b] = true

	arrivedAtRequest := 0

	ifEqual := func(inner_b elevio.ButtonType, inner_f int) {
		if inner_b == b && inner_f == f {
			arrivedAtRequest = 1
		}
	}

	duration := 0

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
