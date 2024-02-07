package elevator

import (
	"mymodule/config"
)

type ElevatorState int

const (
	E_Idle ElevatorState = iota
	E_Moving
	E_Stop
	E_DoorOpen
)

type Elevator struct {
	floor    int
	requests [config.N_FLOORS][config.N_BUTTONS]int
	dir      int
	state
}
