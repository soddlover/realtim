package elevatorFSM

import (
	. "mymodule/config"
	. "mymodule/elevator/elevio"
)

func elevStart(drv_floors chan int, elevator Elev) {
	for floor := 0; floor < N_FLOORS; floor++ {
		for button := 0; button < N_BUTTONS; button++ {
			SetButtonLamp(ButtonType(button), floor, false)
		}
	}
	if GetFloor() == -1 {
		SetMotorDirection(MD_Down)
		<-drv_floors
		SetMotorDirection(MD_Stop)
	}
	SetFloorIndicator(GetFloor())
	elevator.Dir = DirStop
}

func shouldStop(elevator Elev) bool {
	switch elevator.Dir {
	case DirUp:
		return elevator.Queue[elevator.Floor][BT_HallUp] ||
			elevator.Queue[elevator.Floor][BT_Cab] ||
			!ordersAbove(elevator)
	case DirDown:
		return elevator.Queue[elevator.Floor][BT_HallDown] ||
			elevator.Queue[elevator.Floor][BT_Cab] ||
			!ordersBelow(elevator)
	case DirStop:
	default:
	}
	return false
}

func chooseDirection(elevator Elev) ElevatorDirection {
	switch elevator.Dir {
	case DirStop:
		if ordersAbove(elevator) {
			return DirUp
		} else if ordersBelow(elevator) {
			return DirDown
		} else {
			return DirStop
		}
	case DirUp:
		if ordersAbove(elevator) {
			return DirUp
		} else if ordersBelow(elevator) {
			return DirDown
		} else {
			return DirStop
		}

	case DirDown:
		if ordersBelow(elevator) {
			return DirDown
		} else if ordersAbove(elevator) {
			return DirUp
		} else {
			return DirStop
		}
	}
	return DirStop
}

func ordersAbove(elevator Elev) bool {
	for floor := elevator.Floor + 1; floor < N_FLOORS; floor++ {
		for btn := 0; btn < N_BUTTONS; btn++ {
			if elevator.Queue[floor][btn] {
				return true
			}
		}
	}
	return false
}

func ordersBelow(elevator Elev) bool {
	for floor := 0; floor < elevator.Floor; floor++ {
		for btn := 0; btn < N_BUTTONS; btn++ {
			if elevator.Queue[floor][btn] {
				return true
			}
		}
	}
	return false
}
