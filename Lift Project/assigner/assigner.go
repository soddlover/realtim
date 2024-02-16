package assigner

import (
	. "mymodule/config"
	. "mymodule/elevator"
	"mymodule/elevator/elevio"
)

type World struct {
	Map map[string]Elev
}

type OrderAndID struct {
	Order Order
	ID    string
}

func timeToServeRequest(e_old Elev, b elevio.ButtonType, f int) int {
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
		duration += TRAVEL_TIME / 2
		e.Floor += int(e.Dir)
	case EB_DoorOpen:
		duration -= DOOR_OPEN_TIME / 2
	}

	for {
		if ShouldStop(e) {
			e = requestsClearAtCurrentFloor(e, ifEqual)
			if arrivedAtRequest == 1 {
				return duration
			}
			duration += DOOR_OPEN_TIME
			e.Dir = ChooseDirection(e)
		}
		e.Floor += int(e.Dir)
		duration += TRAVEL_TIME
	}
}

func requestsClearAtCurrentFloor(e_old Elev, f func(elevio.ButtonType, int)) Elev {
	e := e_old
	for b := elevio.ButtonType(0); b < N_BUTTONS; b++ {
		if e.Queue[e.Floor][b] {
			e.Queue[e.Floor][b] = false
			if f != nil {
				f(b, e.Floor)
			}
		}
	}
	return e
}

func Assigner(channels Channels, world *World) {

	for {
		select {
		case order := <-channels.OrderRequest:
			if order.Button == elevio.BT_Cab {
				channels.OrderAssigned <- order
				continue
			}
			channels.OrderAssigned <- order

			/*
				for {
					best_id := ""
					best_duration := 1000000
					for id, elevator := range world.Map {
						if elevator.State == Undefined {
							continue
						}
						duration := timeToServeRequest(elevator, order.Button, order.Floor)
						if duration < best_duration {
							best_duration = duration
							best_id = id
						}
					}

					if best_id == self_id {
						channels.OrderAssigned <- order
					}

					peer <- OrderAndID{order, best_id}

					confirmation := <-confirmed
					if confirmation {
						// Confirmation was successful, break the loop to handle the next order
						break
					} else {
						elevator := world.Map[best_id]
						elevator.State = Undefined
						world.Map[best_id] = elevator
						// Confirmation was not successful, remove the failed elevator and try again
					}
				}
			*/
		}
	}
}
