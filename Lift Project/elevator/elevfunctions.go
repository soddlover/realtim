package elevatorFSM

import (
	"fmt"
	. "mymodule/config"
	. "mymodule/elevator/elevio"
	. "mymodule/types"
)

func elevStart(drv_floors <-chan int) {

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
		if elevator.Floor < len(elevator.Queue) &&
			4 > len(elevator.Queue[elevator.Floor]) {
			return elevator.Queue[elevator.Floor][BT_HallUp] ||
				elevator.Queue[elevator.Floor][BT_Cab] || !OrdersAbove(elevator)

		} else {
			fmt.Println("ERROR: Index out of bounds. The floor number is greater than the length of the queue.")
			return false
		}
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

func clearAtFloor(elevator *Elev, orderDelete chan<- Orderstatus) {

	elevator.Queue[elevator.Floor][BT_Cab] = false
	if elevator.Dir == DirUp {
		if elevator.Queue[elevator.Floor][BT_HallUp] {
			elevator.Queue[elevator.Floor][BT_HallUp] = false
			orderDelete <- Orderstatus{Owner: Self_nr, Floor: elevator.Floor, Button: BT_HallUp, Status: true}
		}
		if !OrdersAbove(*elevator) {
			if elevator.Queue[elevator.Floor][BT_HallDown] {
				elevator.Queue[elevator.Floor][BT_HallDown] = false
				orderDelete <- Orderstatus{Owner: Self_nr, Floor: elevator.Floor, Button: BT_HallDown, Status: true}
			}
		}
	} else if elevator.Dir == DirDown {
		if elevator.Queue[elevator.Floor][BT_HallDown] {
			elevator.Queue[elevator.Floor][BT_HallDown] = false
			orderDelete <- Orderstatus{Owner: Self_nr, Floor: elevator.Floor, Button: BT_HallDown, Status: true}
		}
		if !OrdersBelow(*elevator) {
			if elevator.Queue[elevator.Floor][BT_HallUp] {
				elevator.Queue[elevator.Floor][BT_HallUp] = false
				orderDelete <- Orderstatus{Owner: Self_nr, Floor: elevator.Floor, Button: BT_HallUp, Status: true}
			}
		}
	} else if elevator.Dir == DirStop {
		if elevator.Queue[elevator.Floor][BT_HallUp] {
			elevator.Queue[elevator.Floor][BT_HallUp] = false
			orderDelete <- Orderstatus{Owner: Self_nr, Floor: elevator.Floor, Button: BT_HallUp, Status: true}
		}
		if elevator.Queue[elevator.Floor][BT_HallDown] {
			elevator.Queue[elevator.Floor][BT_HallDown] = false
			orderDelete <- Orderstatus{Owner: Self_nr, Floor: elevator.Floor, Button: BT_HallDown, Status: true}
		}
	}
}
