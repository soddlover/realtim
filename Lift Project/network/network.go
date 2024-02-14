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

func BroadcastNetwork(elevStateTx chan Elev, world *assigner.World) {
	//broadcastWorld := &BroadcastWorld{
	// 	Map: make(map[string]BroadcastState),
	// }
	broadcastStateRx := make(chan BroadcastState)
	broadcastStateTx := make(chan BroadcastState)

	go repeater(elevStateTx, broadcastStateTx, "kukk-id")
	go bcast.Transmitter(16569, broadcastStateTx)
	go bcast.Receiver(16569, broadcastStateRx)

	for {
		//update world view
	}
}
func repeater(elevStateTx chan Elev, repeatedElevState chan BroadcastState, elevId string) {
	var lastElev Elev
	ticker := time.NewTicker(500 * time.Millisecond)
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
