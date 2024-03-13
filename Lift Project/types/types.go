package types

import (
	"encoding/json"
	. "mymodule/config"
	"mymodule/elevator/elevio"
	"sync"
	"time"
)

// this is where all universally used types are definedtype ElevatorState ints
type Button int

type Orderstatus struct {
	Owner  string
	Floor  int
	Button elevio.ButtonType
	Served bool
}

type BcastState struct {
	ElevState      Elev
	ID             string
	SequenceNumber int
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

type NetworkOrders struct {
	Orders [N_FLOORS][N_BUTTONS]string
	Mutex  sync.Mutex
}

type NetworkOrdersData struct {
	NetworkOrders [N_FLOORS][N_BUTTONS]string
	TheChosenOne  bool
}

type HeartBeat struct {
	ID   string
	Time time.Time
}
