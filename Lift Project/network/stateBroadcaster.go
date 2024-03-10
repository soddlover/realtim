package network

import (
	"mymodule/config"
	"mymodule/network/bcast"
	. "mymodule/types"
	"time"
)

func updateBroadcastworld(systemState *SystemState, broadcastStateRx <-chan BcastState) {
	bcastSystem := BcastSystem{Map: make(map[string]BcastState)}
	for {
		select {
		case bcastState := <-broadcastStateRx:
			_, existsInSystemState := systemState.Map[bcastState.ID]
			if !existsInSystemState {
				delete(bcastSystem.Map, bcastState.ID)
			}
			existingBcastState, existsInBcastSystem := bcastSystem.Map[bcastState.ID]
			if existsInBcastSystem {
				if bcastState.SequenceNumber > existingBcastState.SequenceNumber {
					bcastSystem.Map[bcastState.ID] = bcastState
					systemState.Map[bcastState.ID] = bcastState.ElevState
				}
			} else {
				bcastSystem.Map[bcastState.ID] = bcastState
				systemState.Map[bcastState.ID] = bcastState.ElevState
			}
		}
	}
}

func StateBroadcaster(elevatorState <-chan Elev, systemState *SystemState, id string) {
	//init bcast world
	//using same world becuse why not?=
	broadcastStateRx := make(chan BcastState)
	broadcastStateTx := make(chan BcastState)

	go repeater(elevatorState, broadcastStateTx, id)
	go bcast.Transmitter(config.Broadcast_state_port, broadcastStateTx)
	go bcast.Receiver(config.Broadcast_state_port, broadcastStateRx)
	go updateBroadcastworld(systemState, broadcastStateRx)
}

func repeater(elevatorState <-chan Elev, broadcastStateTx chan<- BcastState, elevID string) {
	var lastElev Elev
	ticker := time.NewTicker(2000 * time.Millisecond)
	var broadcastState BcastState

	for i := 0; ; i++ {
		select {
		case elev := <-elevatorState:
			lastElev = elev
		case <-ticker.C:
			broadcastState = BcastState{
				ElevState:      lastElev,
				ID:             elevID,
				SequenceNumber: i,
			}
			broadcastStateTx <- broadcastState
		}
	}
}
