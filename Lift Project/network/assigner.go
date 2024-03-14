package network

import (
	"fmt"
	"mymodule/config"
	. "mymodule/elevator"
	"mymodule/elevator/elevio"
	"mymodule/network/SheriffWrangler/sheriff"
	. "mymodule/types"
	"sort"
	"time"
)

func Assigner(
	addToLocalQueue chan<- Order,
	requestSystemState chan<- bool,
	systemState <-chan map[string]Elev,
	assignOrder <-chan Orderstatus,
	writeNetworkOrders chan<- OrderID,
	requestNetworkOrders chan<- bool,
	networkOrders <-chan [config.N_FLOORS][config.N_BUTTONS]string) {
	for {
		select {
		case order := <-assignOrder:
			//channels.OrderAssigned <- order
			if order.Served {
				fmt.Println("Order being deletetet")
				writeNetworkOrders <- OrderID{Floor: order.Floor, Button: order.Button, ID: ""}
				continue
			}

			requestSystemState <- true
			localSystemState := <-systemState
			requestNetworkOrders <- true
			networkOrders := <-networkOrders

			sortedIDs := calculateSortedIDs(localSystemState, networkOrders, Order{Floor: order.Floor, Button: order.Button})
			for _, id := range sortedIDs {
				if id == config.Self_id {
					addToLocalQueue <- Order{Floor: order.Floor, Button: order.Button}
					writeNetworkOrders <- OrderID{Floor: order.Floor, Button: order.Button, ID: id}
					break
				} else {
					success, _ := sheriff.SendOrderMessage(id, order)
					if success {
						writeNetworkOrders <- OrderID{Floor: order.Floor, Button: order.Button, ID: id}
						break
					}
				}
			}
		}
	}
}

func redistributor(
	nodeUnavailabe <-chan string,
	assignOrder chan<- Orderstatus,
	requestSystemState chan<- bool,
	systemState <-chan map[string]Elev,
	requestNetworkOrders chan<- bool,
	networkOrders <-chan [config.N_FLOORS][config.N_BUTTONS]string) {
	for {
		select {
		case peerID := <-nodeUnavailabe:
			fmt.Println("Node is unavailable, redistributing orders")
			requestNetworkOrders <- true
			networkOrders := <-networkOrders
			for floor := 0; floor < config.N_FLOORS; floor++ {
				for button := 0; button < config.N_BUTTONS; button++ {
					if networkOrders[floor][button] == peerID {
						// Send to assigner for reassignment
						assignOrder <- Orderstatus{Floor: floor, Button: elevio.ButtonType(button), Served: false}
					}
				}
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

func calculateSortedIDs(systemState map[string]Elev, networkOrders [config.N_FLOORS][config.N_BUTTONS]string, order Order) []string {
	var durations []IDAndDuration

	for id, elevator := range systemState {
		if elevator.Obstr {
			fmt.Println("Elevator with id: ", id, " is obstructed")
		}
		if elevator.State == EB_UNAVAILABLE {
			continue
		}

		duration := timeToServeRequest(elevator, order.Button, order.Floor)
		durations = append(durations, IDAndDuration{ID: id, Duration: duration})
	}

	sort.Slice(durations, func(i, j int) bool {
		return durations[i].Duration < durations[j].Duration
	})

	assigned := networkOrders[order.Floor][order.Button]
	if assigned != "" {
		if elev, ok := systemState[assigned]; ok {
			if elev.State != EB_UNAVAILABLE {
				// If the order is already assigned to a working elevator, move it to the front of the list
				durations = append([]IDAndDuration{{ID: assigned, Duration: 0}}, durations...)
			}
		}
	}

	var sortedIDs []string
	for _, idAndDuration := range durations {
		sortedIDs = append(sortedIDs, idAndDuration.ID)
	}

	return sortedIDs
}
