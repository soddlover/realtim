package network

import (
	"fmt"
	"mymodule/config"
	. "mymodule/elevator"
	"mymodule/elevator/elevio"
	"mymodule/network/SheriffDeputyWrangler/sheriff"
	. "mymodule/types"
	"time"
)

func redistributor(
	nodeLeftNetwork <-chan string,
	incomingOrder chan<- Orderstatus,
	systemState map[string]Elev,
	networkOrders *NetworkOrders) {
	for {
		select {
		case peerID := <-nodeLeftNetwork:
			networkOrders.Mutex.Lock()
			delete(systemState, peerID)
			fmt.Printf("world.Map: %v\n", systemState)
			fmt.Println("Node left network, redistributing orders")
			fmt.Println("NetworkOrders: ", networkOrders.Orders)
			//check for orders owned by the leaving node

			for floor := 0; floor < len(networkOrders.Orders); floor++ {
				for button := 0; button < len(networkOrders.Orders[button]); button++ {
					if networkOrders.Orders[floor][button] == peerID {
						// Send to assigner for reassignment
						networkOrders.Mutex.Unlock()
						incomingOrder <- Orderstatus{Floor: floor, Button: elevio.ButtonType(button), Served: false, Owner: peerID}
						networkOrders.Mutex.Lock()

					}
				}
			}
			networkOrders.Mutex.Unlock()
		}
	}
}

func Assigner(
	networkUpdate chan<- bool,
	orderAssigned chan<- Order,
	systemState map[string]Elev,
	networkOrders *NetworkOrders,
	nodeLeftNetwork <-chan string,
	incomingOrder chan Orderstatus,
	quitAssigner <-chan bool,
	remainingOrders chan<- [config.N_FLOORS][config.N_BUTTONS]string,
) {
	for {
		select {
		case order := <-incomingOrder:
			//channels.OrderAssigned <- order
			if order.Served {
				fmt.Println("Order being deletetet")
				networkOrders.Mutex.Lock()
				networkOrders.Orders[order.Floor][order.Button] = ""
				networkOrders.Mutex.Unlock()
				networkUpdate <- true
				networkOrders.Mutex.Lock()
				UpdateLightsFromNetworkOrders(networkOrders.Orders)
				networkOrders.Mutex.Unlock()
				continue
			}
			networkOrders.Mutex.Lock()
			best_id := calculateFastestID(systemState, networkOrders, Order{Floor: order.Floor, Button: order.Button})
			order.Owner = best_id
			networkOrders.Orders[order.Floor][order.Button] = best_id
			networkOrders.Mutex.Unlock()
			networkUpdate <- true
			networkOrders.Mutex.Lock()
			UpdateLightsFromNetworkOrders(networkOrders.Orders)
			networkOrders.Mutex.Unlock()
			fmt.Println("Added to NetworkOrders maps")
			if best_id == config.Self_id {
				orderAssigned <- Order{Floor: order.Floor, Button: order.Button}
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

func calculateFastestID(systemState map[string]Elev, networkOrders *NetworkOrders, order Order) string {

	best_id := config.Self_id
	best_duration := 1000000 * time.Second
	for id, elevator := range systemState {
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
	assigned := networkOrders.Orders[order.Floor][order.Button]
	if assigned != "" {
		if elev, ok := systemState[assigned]; ok {
			if !elev.Obstr && !(elev.State == Undefined) {
				//do nothing as its already assigned to a working elevator, could send an additional message to it incase?
				// fmt.Println("Order already assigned to a working elevator")
				// fmt.Println("SOOME PROBLEMS OCCUR HERE MAYBE???")
				best_id = assigned
			}
		}
	}
	return best_id
}
