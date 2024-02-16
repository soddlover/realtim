package main

import (
	"flag"
	"fmt"
	"mymodule/assigner"
	elev "mymodule/elevator"
	orderCom "mymodule/orderCommunication"

	//. "mymodule/network"
	"mymodule/network/localip"

	"time"
)

func main() {
	fmt.Println("main")

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
		Map: make(map[string]elev.Elev),
	}

	channels := elev.Channels{
		ElevatorStates: make(chan elev.Elev),
		OrderRequest:   make(chan elev.Order),
		OrderComplete:  make(chan elev.Order),
		OrderAssigned:  make(chan elev.Order),
	}
	orderCom.TestOrderCommunication(channels, world, id)

	// testchan := make(chan Elev)
	// go SendRandomShitMotherFucker(testchan)
	//fmt.Print("Hello, World!")
	// go RunElev(channels)
	// go StateBroadcaster(testchan, world, id)
	// go PeerConnector(id, world)
	select {}
}

func SendRandomShitMotherFucker(outputChan chan<- elev.Elev) {
	for {
		elevatorState := elev.Elev{
			State: elev.EB_Idle,
			Dir:   elev.DirStop,
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
