package main

import (
	"flag"
	"mymodule/backup"
	"mymodule/config"
	elev "mymodule/elevator"
	"mymodule/network"
	"mymodule/network/localip"
	. "mymodule/types"
)

func main() {
	var id string
	flag.StringVar(&id, "id", "", "id of this peer, default to 0. If running several elevators on same host ID, must be increased for each new elevator.")
	enableWatcher := flag.Bool("enableWatcher", false, " Eneables process pair, watcher") //remove before delivery
	flag.Parse()
	if id == "" {
		id = "0"
	}
	localIP := localip.LocalIP()
	config.SELF_ID = localIP + ":" + id

	initElev := backup.Backup(*enableWatcher)

	elevatorStateBroadcast := make(chan Elev, config.NETWORK_BUFFER_SIZE)
	localOrderRequest := make(chan Order, config.ELEVATOR_BUFFER_SIZE)
	addToLocalQueue := make(chan Order, config.ELEVATOR_BUFFER_SIZE)
	localOrderServed := make(chan Orderstatus, config.ELEVATOR_BUFFER_SIZE)

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
