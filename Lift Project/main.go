package main

import (
	"flag"
	"fmt"
	"mymodule/assigner"
	. "mymodule/elevator"
	. "mymodule/network"
	"time"
)

func main() {

	var id string
	flag.StringVar(&id, "id", "", "id of this peer")
	flag.Parse()

	world := &assigner.World{
		Map: make(map[string]Elev),
	}

	channels := Channels{
		ElevatorStates: make(chan Elev),
		OrderRequest:   make(chan Order),
		OrderComplete:  make(chan Order),
		OrderAssigned:  make(chan Order),
	}
	testchan := make(chan Elev)
	go SendRandomShitMotherFucker(testchan)
	fmt.Print("Hello, World!")
	go RunElev(channels)
	go StateBroadcaster(testchan, world, id)
	go PeerConnector(id, world)
	select {}
}

func SendRandomShitMotherFucker(outputChan chan<- Elev) {
	for {
		elevatorState := Elev{
			State: EB_Idle,
			Dir:   DirStop,
			Floor: 0,
			Queue: [4][3]bool{},
		}
		outputChan <- elevatorState
		time.Sleep(5 * time.Second)
		elevatorState.Floor = 1
		outputChan <- elevatorState
		time.Sleep(5 * time.Second)

	}
}
