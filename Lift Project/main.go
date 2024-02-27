package main

import (
	"flag"
	"fmt"
	"mymodule/backup"
	"mymodule/config"
	. "mymodule/elevator"
	"mymodule/network"
	"mymodule/network/localip"
	"time"
	//. "mymodule/network"
)

func main() {
	// WHen starting
	var id string
	flag.StringVar(&id, "id", "", "id of this peer")
	fresh := flag.Bool("fresh", false, "Start a fresh elevator")
	flag.Parse()
	localIP, err := localip.LocalIP()
	if err != nil {
		fmt.Println(err)
		localIP = "DISCONNECTED"
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

	world := &network.World{
		Map: make(map[string]Elev),
	}

	channels := Channels{
		ElevatorStates:          make(chan Elev, 10),
		ElevatorStatesBroadcast: make(chan Elev, 10),
		OrderRequest:            make(chan Order, 10),
		OrderComplete:           make(chan Order, 10),
		OrderAssigned:           make(chan Orderstatus, 10),
		OrderDelete:             make(chan Orderstatus, 10),
	}

	//fmt.Print("Hello, World!")
	// go RunElev(channels)
	go network.PeerConnector(id, world, channels)
	go network.StateBroadcaster(channels.ElevatorStatesBroadcast, world, id)
	// go PeerConnector(id, world)
	go backup.WriteBackup(channels.ElevatorStates)
	go RunElev(channels, initElev)
	//go Assigner(channels, world)
	//go printWorld(world)
	select {}
}

func printWorld(world *network.World) {
	ticker := time.NewTicker(10 * time.Second)
	for {
		select {
		case <-ticker.C:
			for _, value := range world.Map {
				//fmt.Println("Key:", key, "Value:", value)
				fmt.Println("Direction:", value.Dir)
				fmt.Println("Floor:", value.Floor)
				//fmt.Println("Queue:", value.Queue)
				fmt.Println("State:", value.State)
			}
		}
	}
}
