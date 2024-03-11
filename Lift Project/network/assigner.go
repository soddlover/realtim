package network

import (
	"fmt"
	"mymodule/config"
	. "mymodule/elevator"
	"mymodule/elevator/elevio"
	"mymodule/network/SheriffDeputyWrangler/sheriff"
	"mymodule/network/SheriffDeputyWrangler/wrangler"
	. "mymodule/types"
	"time"
)

// func redistributor(nodeLeftNetwork <-chan string, incomingOrder chan<- Orderstatus, world *World, NetworkOrders *[config.N_FLOORS][config.N_BUTTONS]string) {
// 	for {
// 		select {
// 		case peerID := <-nodeLeftNetwork:
// 			delete(world.Map, peerID)
// 			fmt.Printf("world.Map: %v\n", world.Map)
// 			fmt.Println("Node left network, redistributing orders")
// 			fmt.Println("NetworkOrders: ", NetworkOrders)
// 			//check for orders owned by the leaving node
// 			for floor := 0; floor < len(NetworkOrders); floor++ {
// 				for button := 0; button < len(NetworkOrders[button]); button++ {
// 					if NetworkOrders[floor][button] == peerID {
// 						// Send to assigner for reassignment
// 						incomingOrder <- Orderstatus{Floor: floor, Button: elevio.ButtonType(button), Status: false, Owner: peerID}
// 					}
// 				}
// 			}
// 		}
// 	}
// }

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
			if state == st_sherriff {
				incomingOrder <- orderstat
			} else {
				wrangler.SendOrderToSheriff(orderstat)
			}
		case orderstat := <-orderDelete:
			if state == st_sherriff {
				incomingOrder <- orderstat
			} else {
				wrangler.SendOrderToSheriff(orderstat)
			}
		}
	}
}

func Assigner(
	networkUpdate chan<- bool,
	orderAssigned chan<- Orderstatus,
	systemstate *SystemState,
	networkOrders *[config.N_FLOORS][config.N_BUTTONS]string,
	nodeLeftNetwork <-chan string,
	incomingOrder chan Orderstatus,
	quitAssigner <-chan bool,
	remainingOrders chan<- [config.N_FLOORS][config.N_BUTTONS]string) {

	for {
		select {
		case order := <-incomingOrder:
			//channels.OrderAssigned <- order
			if order.Served {
				fmt.Println("Order being deletetet")
				networkOrders[order.Floor][order.Button] = ""
				networkUpdate <- true
				continue
			}

			best_id := config.Self_id
			best_duration := 1000000 * time.Second
			for id, elevator := range systemstate.Map {
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
			assigned := networkOrders[order.Floor][order.Button]
			if assigned != "" {
				if elev, ok := systemstate.Map[assigned]; ok {
					if !elev.Obstr && !(elev.State == Undefined) {
						//do nothing as its already assigned to a working elevator, could send an additional message to it incase?
						fmt.Println("Order already assigned to a working elevator")
						fmt.Println("SOOME PROBLEMS OCCUR HERE MAYBE???")
						best_id = assigned
					}
				}
			}
			order.Owner = best_id
			networkOrders[order.Floor][order.Button] = best_id
			networkUpdate <- true
			fmt.Println("Added to NetworkOrders maps")
			if best_id == config.Self_id {
				orderAssigned <- order
			} else {
				go sheriff.SendOrderMessage(best_id, order)
			}

		case peerID := <-nodeLeftNetwork:
			delete(systemstate.Map, peerID)
			fmt.Printf("world.Map: %v\n", systemstate.Map)
			fmt.Println("Node left network, redistributing orders")
			fmt.Println("NetworkOrders: ", networkOrders)
			//check for orders owned by the leaving node
			for floor := 0; floor < len(networkOrders); floor++ {
				for button := 0; button < len(networkOrders[button]); button++ {
					if networkOrders[floor][button] == peerID {
						// Send to assigner for reassignment
						incomingOrder <- Orderstatus{Floor: floor, Button: elevio.ButtonType(button), Served: false, Owner: peerID}
					}
				}
			}
		case <-quitAssigner:
			remainingOrders <- *networkOrders
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
