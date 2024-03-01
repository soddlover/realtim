package config

const (
	N_FLOORS    = 4
	N_ELEVATORS = 1
	N_BUTTONS   = 3
)

const TRAVEL_TIME = 2
const DOOR_OPEN_TIME = 3
const Sheriff_port = 20000
const Broadcast_state_port = 16569
const Peer_port = 15647
const TCP_port = 16000
const Sheriff_deputy_port = 16001

type Button int

var Self_id string = ""
var Self_nr string = "0"

//this really should not be here right?
