package types

import (
	"encoding/json"
	"mymodule/config"
	"time"
)

type ButtonType int

type Orderstatus struct {
	Floor  int
	Button ButtonType
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
	EB_UNAVAILABLE
)
const (
	BT_HallUp   ButtonType = 0
	BT_HallDown            = 1
	BT_Cab                 = 2
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
	Queue [config.N_FLOORS][config.N_BUTTONS]bool
}

type Order struct {
	Floor  int
	Button ButtonType
}

type OrderID struct {
	Floor  int
	Button ButtonType
	ID     string
}

type Message struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

type NetworkOrdersData struct {
	NetworkOrders [config.N_FLOORS][config.N_BUTTONS]string
	TheChosenOne  string
}

type NetworkOrderPacket struct {
	NetworkOrders [config.N_FLOORS][config.N_BUTTONS]string
	TheChosenOne  string
	SequenceNum   int
}

type HeartBeat struct {
	ID   string
	Time time.Time
}

type IDAndDuration struct {
	ID       string
	Duration time.Duration
}
