package main

import (
	"flag"
	"fmt"
	"mymodule/assigner"
	. "mymodule/elevator"

	//. "mymodule/network"
	"mymodule/network/localip"

	"time"
)

func main() {
	fmt.Println("main running.")

	var id string
	flag.StringVar(&id, "id", "", "id of this peer")
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

	world := &assigner.World{
		Map: make(map[string]Elev),
	}

	channels := Channels{
		ElevatorStates: make(chan Elev),
		OrderRequest:   make(chan Order),
		OrderComplete:  make(chan Order),
		OrderAssigned:  make(chan Order),
	}
	assigner.TestOrderCommunication(channels, world, id)

	// testchan := make(chan Elev)
	// go SendRandomShitMotherFucker(testchan)
	//fmt.Print("Hello, World!")
	// go RunElev(channels)
	// go StateBroadcaster(testchan, world, id)
	// go PeerConnector(id, world)
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
