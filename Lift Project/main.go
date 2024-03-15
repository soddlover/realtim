package main

import (
	"Project/backup"
	. "Project/config"
	elev "Project/elevator"
	"Project/network"
	"Project/network/localip"
	. "Project/types"
	"flag"
)

func main() {
	var id string
	flag.StringVar(&id, "id", "", "id of this peer, default to 0. If running several elevators on same host ID, must be increased for each new elevator.")
	disableWatcher := flag.Bool("disableWatcher", false, " Eneables process pair, watcher") //remove before delivery
	flag.Parse()
	if id == "" {
		id = "0"
	}
	localIP := localip.LocalIP()
	SELF_ID = localIP + ":" + id

	initElev := backup.Backup(*disableWatcher)

	elevatorStateBroadcast := make(chan Elev, NETWORK_BUFFER_SIZE)
	localOrderRequest := make(chan Order, ELEVATOR_BUFFER_SIZE)
	addToLocalQueue := make(chan Order, ELEVATOR_BUFFER_SIZE)
	localOrderServed := make(chan Orderstatus, ELEVATOR_BUFFER_SIZE)

	go elev.RunElev(
		elevatorStateBroadcast,
		localOrderRequest,
		addToLocalQueue,
		localOrderServed,
		initElev)

	go network.NetworkFSM(
		elevatorStateBroadcast,
		localOrderRequest,
		addToLocalQueue,
		localOrderServed)

	select {}
}
