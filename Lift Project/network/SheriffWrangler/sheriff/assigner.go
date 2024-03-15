package sheriff

import (
	. "Project/config"
	"Project/elevator"
	. "Project/types"
	"fmt"
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
	networkOrders <-chan [N_FLOORS][N_BUTTONS]string) {

	for {

		order := <-assignOrder
		if order.Served {
			writeNetworkOrders <- OrderID{Floor: order.Floor, Button: order.Button, ID: ""}
			continue
		}
		requestSystemState <- true
		localSystemState := <-systemState

		requestNetworkOrders <- true
		networkOrders := <-networkOrders
		assigned := false
		sortedIDs := calculateSortedIDs(localSystemState, networkOrders, Order{Floor: order.Floor, Button: order.Button})
		for _, id := range sortedIDs {
			if id == SELF_ID {
				addToLocalQueue <- Order{Floor: order.Floor, Button: order.Button}
				writeNetworkOrders <- OrderID{Floor: order.Floor, Button: order.Button, ID: id}
				fmt.Println("Order assigned to self", OrderID{Floor: order.Floor, Button: order.Button, ID: id})
				assigned = true
				break
			} else {
				success, _ := SendOrderMessage(id, order)
				if success {
					writeNetworkOrders <- OrderID{Floor: order.Floor, Button: order.Button, ID: id}
					assigned = true
					fmt.Println("Order succesfully sent and assigned to ", OrderID{Floor: order.Floor, Button: order.Button, ID: id})
					break
				}
			}
		}
		if !assigned {
			fmt.Println("No available elevators attempting assigning again")
			addToLocalQueue <- Order{Floor: order.Floor, Button: order.Button}
			writeNetworkOrders <- OrderID{Floor: order.Floor, Button: order.Button, ID: SELF_ID}

		}

	}
}

func redistributor(
	nodeUnavailabe <-chan string,
	assignOrder chan<- Orderstatus,
	requestNetworkOrders chan<- bool,
	networkOrders <-chan [N_FLOORS][N_BUTTONS]string) {

	for {
		peerID := <-nodeUnavailabe
		fmt.Println("Node is unavailable, redistributing orders")
		requestNetworkOrders <- true
		networkOrders := <-networkOrders
		for floor, floorOrders := range networkOrders {
			for button, id := range floorOrders {
				if id == peerID {
					assignOrder <- Orderstatus{Floor: floor, Button: ButtonType(button), Served: false}
				}
			}
		}

	}
}

func timeToServeRequest(e_old Elev, b ButtonType, f int) time.Duration {

	e := e_old
	e.Queue[f][b] = true

	arrivedAtRequest := 0

	ifEqual := func(inner_b ButtonType, inner_f int) {
		if inner_b == b && inner_f == f {
			arrivedAtRequest = 1
		}
	}

	duration := 0 * time.Second

	switch e.State {
	case EB_Idle:
		e.Dir = elevator.ChooseDirection(e)
		if e.Dir == DirStop {
			return duration
		}
	case EB_Moving:
		duration += TRAVEL_TIME / 2
		e.Floor += int(e.Dir)
	case EB_DoorOpen:
		duration -= DOOR_OPEN_TIME / 2
		if !elevator.OrdersAbove(e) && !elevator.OrdersBelow(e) {
			return duration
		}
	}

	for {
		if elevator.ShouldStop(e) {
			e = requestsClearAtCurrentFloor(e, ifEqual)
			if arrivedAtRequest == 1 {
				return duration
			}
			duration += DOOR_OPEN_TIME
			e.Dir = elevator.ChooseDirection(e)
		}
		e.Floor += int(e.Dir)
		duration += TRAVEL_TIME
	}
}

func requestsClearAtCurrentFloor(e_old Elev, f func(ButtonType, int)) Elev {

	e := e_old
	for b := ButtonType(0); b < N_BUTTONS; b++ {
		if e.Queue[e.Floor][b] {
			e.Queue[e.Floor][b] = false
			if f != nil {
				f(b, e.Floor)
			}
		}
	}
	return e
}

func calculateSortedIDs(
	systemState map[string]Elev,
	networkOrders [N_FLOORS][N_BUTTONS]string,
	order Order) []string {

	var durations []IDAndDuration

	for id, elevator := range systemState {

		if elevator.State == EB_Unavailable {
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
			if elev.State != EB_Unavailable {
				// If the order is already assigned to a working elevator, move it to the front of the list
				durations = append([]IDAndDuration{{ID: assigned, Duration: 0}}, durations...)
				fmt.Println("Already assigned to elevator, moving to front of list", assigned)
			}
		}
	}

	var sortedIDs []string
	for _, idAndDuration := range durations {
		sortedIDs = append(sortedIDs, idAndDuration.ID)
	}

	return sortedIDs
}
