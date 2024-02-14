package network

import (
	"mymodule/assigner"
	. "mymodule/elevator"
	"mymodule/network/bcast"
	"time"
)

type BroadcastState struct {
	ElevState      Elev
	Id             string
	SequenceNumber int
}

type BroadcastWorld struct {
	Map map[string]BroadcastState
}

func updateBroadcastworld(bcastWorld BroadcastWorld, broadcastStateRx chan BroadcastState) {
	for {
		//update world view
		select {
		case bcastState := <-broadcastStateRx:
			// elev := bcastState.ElevState
			if _, ok := bcastWorld.Map[bcastState.Id]; ok {
				if bcastState.SequenceNumber > bcastWorld.Map[bcastState.Id].SequenceNumber {
					bcastWorld.Map[bcastState.Id] = bcastState
					// fmt.Println("Updated value")
					// fmt.Printf("%+v\n", bcastWorld)
				}
			} else {
				//might be unnecicary if implemented by peer functionality.
				bcastWorld.Map[bcastState.Id] = bcastState
				//fmt.Println("Added new element to map.")
			}
		}
	}
}

func StateBroadcaster(elevStateTx chan Elev, world *assigner.World, id string) {
	//init bcast world
	bcastWorld := BroadcastWorld{Map: make(map[string]BroadcastState)}

	broadcastStateRx := make(chan BroadcastState)
	broadcastStateTx := make(chan BroadcastState)

	go repeater(elevStateTx, broadcastStateTx, id)
	go bcast.Transmitter(16569, broadcastStateTx)
	go bcast.Receiver(16569, broadcastStateRx)
	go updateBroadcastworld(bcastWorld, broadcastStateRx)
}

func repeater(elevStateTx chan Elev, repeatedElevState chan BroadcastState, elevId string) {
	var lastElev Elev
	ticker := time.NewTicker(2000 * time.Millisecond)
	var broadcastState BroadcastState

	for i := 0; ; i++ {
		select {
		case elev := <-elevStateTx:
			lastElev = elev
		case <-ticker.C:
			broadcastState = BroadcastState{lastElev, elevId, i}
			// .ElevState = lastElev
			// broadcastState.Id = elevId
			// broadcastState.SequenceNumber = i
			repeatedElevState <- broadcastState
		}
	}
}
