package config

import "time"

const (
	N_FLOORS             = 9
	N_ELEVATORS          = 1
	N_BUTTONS            = 3
	TRAVEL_TIME          = 2
	DOOR_OPEN_TIME       = 3 * time.Second
	MOTOR_ERROR_TIME     = 3 * time.Second
	Sheriff_port         = 20000
	Broadcast_state_port = 16569
	Peer_port            = 15647
	TCP_port             = 16000
	Sheriff_deputy_port  = 16001
	SimulatorPort        = 15657
)

type Button int

var Self_id string = ""
var Self_nr string = "0"

//this really should not be here right?
