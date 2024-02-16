package main

import (
	"flag"
	"fmt"
	. "mymodule/assigner"
	"mymodule/backup"
	"mymodule/config"
	. "mymodule/elevator"
	"mymodule/network"
	"mymodule/network/localip"
	"time"
	//. "mymodule/network"
)

func main() {

	var id string
	flag.StringVar(&id, "id", "", "id of this peer")
	fresh := flag.Bool("fresh", false, "Start a fresh elevator")
	flag.Parse()

	if id == "" {
		localIP, err := localip.LocalIP()
		if err != nil {
			fmt.Println(err)
			localIP = "DISCONNECTED"
		}
		//id = fmt.Sprintf("peer-%s-%d", localIP, os.Getpid())
		id = localIP
	}
	config.Self_id = id
	initElev := backup.Backup(*fresh)

	if (initElev == Elev{}) {
		fmt.Println("Starting with fresh elevator")

	}

	world := &World{
		Map: make(map[string]Elev),
	}

	channels := Channels{
		ElevatorStates:          make(chan Elev, 10),
		ElevatorStatesBroadcast: make(chan Elev, 10),
		OrderRequest:            make(chan Order, 10),
		OrderComplete:           make(chan Order, 10),
		OrderAssigned:           make(chan Order, 10),
	}

	//fmt.Print("Hello, World!")
	// go RunElev(channels)
	go network.PeerConnector(id, world)
	go network.StateBroadcaster(channels.ElevatorStatesBroadcast, world, id)
	// go PeerConnector(id, world)
	go backup.WriteBackup(channels.ElevatorStates)
	go RunElev(channels, initElev)
	go Assigner(channels, world)
	go printWorld(world)
	select {}
}

func printWorld(world *World) {
	ticker := time.NewTicker(5 * time.Second)
	for {
		select {
		case <-ticker.C:
			for key, value := range world.Map {
				fmt.Println("Key:", key, "Value:", value)
			}
		}
	}
}
