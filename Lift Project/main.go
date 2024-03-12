package main

import (
	"flag"
	"fmt"
	"mymodule/backup"
	"mymodule/config"
	elev "mymodule/elevator"
	"mymodule/network"
	"mymodule/network/localip"
	. "mymodule/types"
	"time"
)

func main() {

	// WHen starting
	var id string
	flag.StringVar(&id, "id", "", "id of this peer")
	fresh := flag.Bool("fresh", false, "Start a fresh elevator")
	flag.Parse()

	var localIP string
	var err error
	for {
		localIP, err = localip.LocalIP()
		if err != nil {
			fmt.Println(err)
			time.Sleep(1 * time.Second)
		} else {
			break
		}
	}

	if id == "" {
		id = localIP + ":0"
		config.Self_nr = "0"
	} else {
		config.Self_nr = id
		id = localIP + ":" + id
	}

	config.Self_id = id

	initElev := backup.Backup(*fresh)

	if (initElev == Elev{}) {
		fmt.Println("Starting with fresh elevator")
	}

	systemState := make(map[string]Elev)

	elevatorStateBackup := make(chan Elev, 10)
	elevatorStateBroadcast := make(chan Elev, 10)
	orderRequest := make(chan Order, 10)
	//orderComplete := make(chan Order, 10)
	orderAssigned := make(chan Order, 10)
	orderDelete := make(chan Orderstatus, 10)
	incomingOrder := make(chan Orderstatus, 10)

	go network.StateBroadcaster(elevatorStateBroadcast, systemState, id)
	// go PeerConnector(id, systemState)
	go backup.WriteBackup(elevatorStateBackup)
	go elev.RunElev(
		elevatorStateBackup,
		elevatorStateBroadcast,
		orderRequest,
		orderAssigned,
		orderDelete,
		initElev)
	go network.NetworkFSM(
		orderRequest,
		orderAssigned,
		orderDelete,
		systemState,
		incomingOrder)

	//go Assigner(channels, systemState)
	//go printsystemState(systemState)
	select {}
}
