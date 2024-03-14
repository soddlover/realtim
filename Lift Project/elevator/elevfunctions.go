package elevatorFSM

import (
	"fmt"
	"mymodule/config"
	"mymodule/elevator/elevio"
	. "mymodule/types"
	"time"
)

const N_FLOORS = config.N_FLOORS
const N_BUTTONS = config.N_BUTTONS

func elevatorInit(elevator Elev, drv_floors <-chan int) Elev {
	if (elevator == Elev{}) {
		elevator = Elev{
			State: EB_Idle,
			Dir:   DirStop,
			Floor: elevio.GetFloor(),
			Queue: [N_FLOORS][N_BUTTONS]bool{},
			Obstr: false,
		}
	}
	for floor := 0; floor < N_FLOORS; floor++ {
		elevio.SetButtonLamp(elevio.BT_HallUp, floor, false)
		elevio.SetButtonLamp(elevio.BT_HallDown, floor, false)
	}
	if elevio.GetFloor() == -1 {
		elevio.SetMotorDirection(elevio.MD_Down)
		ticker := time.NewTicker(config.MOTOR_ERROR_TIME)
		defer ticker.Stop()
		select {
		case <-drv_floors:
			elevio.SetMotorDirection(elevio.MD_Stop)
			ticker.Stop()
			elevio.SetFloorIndicator(elevio.GetFloor())
			fmt.Println("Arrived at floor: ", elevio.GetFloor())
			elevator.State = EB_Idle
			elevator.Floor = elevio.GetFloor()
			elevator.Dir = ChooseDirection(elevator)
			elevio.SetMotorDirection(elevio.MotorDirection(elevator.Dir))

		case <-ticker.C:
			fmt.Println("Failed to arrive at floor within time limit, killing myself")
			elevator.State = EB_UNAVAILABLE
			ticker.Stop()
		}
	}
	return elevator
}

func ShouldStop(elevator Elev) bool {
	switch elevator.Dir {
	case DirUp:
		if elevator.Floor < len(elevator.Queue) &&
			4 > len(elevator.Queue[elevator.Floor]) {
			return elevator.Queue[elevator.Floor][elevio.BT_HallUp] ||
				elevator.Queue[elevator.Floor][elevio.BT_Cab] || !OrdersAbove(elevator)

		} else {
			fmt.Println("ERROR: Index out of bounds. The floor number is greater than the length of the queue.")
			return false
		}
	case DirDown:
		return elevator.Queue[elevator.Floor][elevio.BT_HallDown] ||
			elevator.Queue[elevator.Floor][elevio.BT_Cab] ||
			!OrdersBelow(elevator)
	case DirStop:
		return true //??
	default:
		return true
	}
}

func ChooseDirection(elevator Elev) ElevatorDirection {
	switch elevator.Dir {
	case DirStop:
		if OrdersAbove(elevator) {
			return DirUp
		} else if OrdersBelow(elevator) {
			return DirDown
		} else {
			return DirStop
		}
	case DirUp:
		if OrdersAbove(elevator) {
			return DirUp
		} else if OrdersBelow(elevator) {
			return DirDown
		} else {
			return DirStop
		}

	case DirDown:
		if OrdersBelow(elevator) {
			return DirDown
		} else if OrdersAbove(elevator) {
			return DirUp
		} else {
			return DirStop
		}
	default:
		return DirStop
	}
}

func OrdersAbove(elevator Elev) bool {
	for floor := elevator.Floor + 1; floor < N_FLOORS; floor++ {
		for btn := 0; btn < N_BUTTONS; btn++ {
			if elevator.Queue[floor][btn] {
				return true
			}
		}
	}
	return false
}

func OrdersBelow(elevator Elev) bool {
	for floor := 0; floor < elevator.Floor; floor++ {
		for btn := 0; btn < N_BUTTONS; btn++ {
			if elevator.Queue[floor][btn] {
				return true
			}
		}
	}
	return false
}

func clearAtFloor(elevator *Elev, orderDelete chan<- Orderstatus) {
	elevator.Queue[elevator.Floor][elevio.BT_Cab] = false
	if elevator.Dir == DirUp && elevator.Queue[elevator.Floor][elevio.BT_HallUp] {
		elevator.Queue[elevator.Floor][elevio.BT_HallUp] = false
		orderDelete <- Orderstatus{Floor: elevator.Floor, Button: elevio.BT_HallUp, Served: true}
	} else if elevator.Dir == DirDown && elevator.Queue[elevator.Floor][elevio.BT_HallDown] {
		elevator.Queue[elevator.Floor][elevio.BT_HallDown] = false
		orderDelete <- Orderstatus{Floor: elevator.Floor, Button: elevio.BT_HallDown, Served: true}
	} else if elevator.Dir == DirStop || elevator.Floor == 0 || elevator.Floor == N_FLOORS-1 {
		if elevator.Queue[elevator.Floor][elevio.BT_HallUp] {
			elevator.Queue[elevator.Floor][elevio.BT_HallUp] = false
			orderDelete <- Orderstatus{Floor: elevator.Floor, Button: elevio.BT_HallUp, Served: true}
		}
		if elevator.Queue[elevator.Floor][elevio.BT_HallDown] {
			elevator.Queue[elevator.Floor][elevio.BT_HallDown] = false
			orderDelete <- Orderstatus{Floor: elevator.Floor, Button: elevio.BT_HallUp, Served: true}
		}
	}
}

func updateLights(elevator *Elev) {
	for {
		for floor := 0; floor < N_FLOORS; floor++ {
			for button := 0; button < N_BUTTONS; button++ {
				if elevio.ButtonType(button) == elevio.BT_Cab {
					elevio.SetButtonLamp(elevio.ButtonType(button), floor, elevator.Queue[floor][button])
				}
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func UpdateLightsFromNetworkOrders(networkorders [N_FLOORS][N_BUTTONS]string) {
	for floor := 0; floor < N_FLOORS; floor++ {
		for button := 0; button < N_BUTTONS; button++ {
			if elevio.ButtonType(button) != elevio.BT_Cab {
				if networkorders[floor][button] != "" {
					elevio.SetButtonLamp(elevio.ButtonType(button), floor, true)
				} else {
					elevio.SetButtonLamp(elevio.ButtonType(button), floor, false)
				}
			}
		}
	}
}
