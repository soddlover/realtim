package config

import "time"

const (
	N_FLOORS                  = 4
	N_ELEVATORS               = 3
	N_BUTTONS                 = 3
	TRAVEL_TIME               = 2
	DOOR_OPEN_TIME            = 3 * time.Second
	MOTOR_ERROR_TIME          = 3 * time.Second
	SHERIFF_TRANSMITT_IP_PORT = 20000
	BROADCAST_STATE_PORT      = 16569
	TCP_PORT                  = 16000
	SIMULATOR_PORT            = 15657
	UDP_NETWORK_ORDERS_PORT   = 16568
	HEARTBEAT                 = 20 * time.Millisecond
	HEARTBEAT_DEADLINE        = 2000 * time.Millisecond
	BACKUP_INTERVAL           = 1 * time.Second
	BACKUP_DEADLINE           = 2 * time.Second
	SHERIFF_IP_DEADLINE       = 1 * time.Second
	ELEVATOR_BUFFER_SIZE      = N_FLOORS * N_BUTTONS
	NETWORK_BUFFER_SIZE       = ELEVATOR_BUFFER_SIZE * N_ELEVATORS
	ORDER_DEADLINE            = TRAVEL_TIME * N_FLOORS * time.Second
	NETWORK_ORDER_FREQUENCY   = 200 * time.Millisecond
)

var SELF_ID string = ""
