package types

import (
	"encoding/json"
	. "mymodule/config"
	"mymodule/elevator/elevio"
)

//this is where all universally used types are definedtype ElevatorState int

type Channels struct {
	ElevatorStates          chan Elev
	ElevatorStatesBroadcast chan Elev
	OrderRequest            chan Order
	OrderComplete           chan Order
	OrderAssigned           chan Orderstatus
	OrderDelete             chan Orderstatus
	IncomingOrder           chan Orderstatus
}

type Orderstatus struct {
	Owner   string
	OrderID string
	Floor   int
	Button  elevio.ButtonType
	Status  bool
}

type ElevatorState int

const (
	EB_Idle ElevatorState = iota
	EB_Moving
	EB_DoorOpen
	Undefined
)

type ElevatorDirection int

const (
	DirUp   ElevatorDirection = 1
	DirDown ElevatorDirection = -1
	DirStop ElevatorDirection = 0
)

type Elev struct {
	State ElevatorState
	Dir   ElevatorDirection
	Floor int
	Queue [N_FLOORS][N_BUTTONS]bool
	Obstr bool
}

type Order struct {
	Floor  int
	Button elevio.ButtonType
}

type Message struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}
