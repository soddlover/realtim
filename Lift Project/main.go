package main

import (
	"flag"
	"fmt"
	. "mymodule/assigner"
	"mymodule/backup"
	. "mymodule/elevator"
)

func main() {

	fresh := flag.Bool("fresh", false, "Start a fresh elevator")
	flag.Parse()

	initElev := backup.Backup(*fresh)

	if (initElev == Elev{}) {
		fmt.Println("Starting with fresh elevator")

	}

	world := &World{
		Map: make(map[string]Elev),
	}

	channels := Channels{
		ElevatorStates: make(chan Elev, 10),
		OrderRequest:   make(chan Order, 10),
		OrderComplete:  make(chan Order, 10),
		OrderAssigned:  make(chan Order, 10),
	}
	fmt.Print("Hello, World!")
	go backup.WriteBackup(channels.ElevatorStates)
	go RunElev(channels, initElev)
	go Assigner(channels, world)
	select {}
}
