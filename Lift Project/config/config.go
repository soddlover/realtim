package config

import "time"

const (
	N_FLOORS             = 4
	N_ELEVATORS          = 1
	N_BUTTONS            = 3
	TRAVEL_TIME          = 2
	DOOR_OPEN_TIME       = 3 * time.Second
	MOTOR_ERROR_TIME     = 3 * time.Second
	Sheriff_port         = 20000
	Broadcast_state_port = 16569
	Peer_port            = 15647
	TCP_port             = 16000
	SimulatorPort        = 15657
	HEARTBEAT            = 20 * time.Millisecond
	HEARTBEAT_DEADLINE   = 1000 * time.Millisecond
	BACKUP_INTERVAL      = 1 * time.Second
	BACKUP_DEADLINE      = 2 * time.Second
	SHERIFF_IP_DEADLINE  = 1 * time.Second
)

var Self_id string = ""
var Self_nr string = "0"

//this really should not be here right?
