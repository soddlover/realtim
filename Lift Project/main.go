package main

import (
	"flag"
	"fmt"
	. "mymodule/assigner"
	"mymodule/backup"
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
	assigner.TestOrderCommunication(channels, world, id)

	// testchan := make(chan Elev)
	// go SendRandomShitMotherFucker(testchan)
	//fmt.Print("Hello, World!")
	// go RunElev(channels)
	// go StateBroadcaster(testchan, world, id)
	// go PeerConnector(id, world)
	fmt.Print("Hello, World!")
	go backup.WriteBackup(channels.ElevatorStates)
	go RunElev(channels, initElev)
	go Assigner(channels, world)
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
