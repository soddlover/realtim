package elevatorFSM

import (
	"fmt"
	. "mymodule/config"
	. "mymodule/elevator/elevio"
	. "mymodule/types"
)

func elevStart(drv_floors chan int) {

	if GetFloor() == -1 {
		SetMotorDirection(MD_Down)
		<-drv_floors
		SetMotorDirection(MD_Stop)
	}
	SetFloorIndicator(GetFloor())
	fmt.Println("Arrived at floor: ", GetFloor())
}

func ShouldStop(elevator Elev) bool {
	switch elevator.Dir {
	case DirUp:
		return elevator.Queue[elevator.Floor][BT_HallUp] ||
			elevator.Queue[elevator.Floor][BT_Cab] ||
			!OrdersAbove(elevator)
	case DirDown:
		return elevator.Queue[elevator.Floor][BT_HallDown] ||
			elevator.Queue[elevator.Floor][BT_Cab] ||
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

func clearAtFloor(elevator *Elev, channels Channels) {
	elevator.Queue[elevator.Floor][BT_Cab] = false
	if elevator.Dir == DirUp {
		elevator.Queue[elevator.Floor][BT_HallUp] = false
		channels.OrderComplete <- Order{Floor: elevator.Floor, Button: BT_HallUp}
		if !OrdersAbove(*elevator) {
			elevator.Queue[elevator.Floor][BT_HallDown] = false
			channels.OrderComplete <- Order{Floor: elevator.Floor, Button: BT_HallDown}
		}
	} else if elevator.Dir == DirDown {
		elevator.Queue[elevator.Floor][BT_HallDown] = false
		channels.OrderComplete <- Order{Floor: elevator.Floor, Button: BT_HallDown}
		if !OrdersBelow(*elevator) {
			elevator.Queue[elevator.Floor][BT_HallUp] = false
			channels.OrderComplete <- Order{Floor: elevator.Floor, Button: BT_HallUp}
		}
	}
}
