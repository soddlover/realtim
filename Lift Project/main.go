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
	flag.StringVar(&id, "id", "", "id of this peer")
	fresh := flag.Bool("fresh", false, "Start a fresh elevator") //remove before delivery
	flag.Parse()
	if id == "" {
		id = "0"

	}
	localIP := localip.LocalIP()

	config.Id = localIP + ":" + id

	initElev := backup.Backup(*fresh)

	elevatorStateBackup := make(chan Elev, config.ELEVATOR_BUFFER_SIZE)
	elevatorStateBroadcast := make(chan Elev, config.NETWORK_BUFFER_SIZE)
	localOrderRequest := make(chan Order, config.ELEVATOR_BUFFER_SIZE)
	addToLocalQueue := make(chan Order, config.ELEVATOR_BUFFER_SIZE)
	localOrderServed := make(chan Orderstatus, config.ELEVATOR_BUFFER_SIZE)

	go backup.WriteBackup(
		elevatorStateBackup)

	go elev.RunElev(
		elevatorStateBackup,
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
