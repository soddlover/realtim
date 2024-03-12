package network

import (
	"fmt"
	"mymodule/config"
	"mymodule/network/bcast"
	. "mymodule/types"
	"time"
)

var lastHeartbeat = make(map[string]time.Time)

func CheckHeartbeats(lostConn chan string) {
	lastReported := make(map[string]bool)
	for {
		time.Sleep(1 * time.Second) // Check every 10 seconds

		for id, last := range lastHeartbeat {
			if time.Since(last) > config.HEARTBEAT_DEADLINE { // If it's been more than 30 seconds since the last heartbeat

				if !lastReported[id] {
					fmt.Printf("No heartbeat from %s for more than 3 seconds\n", id)
					lostConn <- id
					lastReported[id] = true
				}

				// Here you can add code to close the connection to the peer
			} else {
				lastReported[id] = false
			}
		}
	}
}

func updateBroadcastworld(systemState map[string]Elev, broadcastStateRx <-chan BcastState) {
	bcastSystem := make(map[string]BcastState)
	for {
		select {
		case bcastState := <-broadcastStateRx:
			lastHeartbeat[bcastState.ID] = time.Now()
			_, existsInSystemState := systemState[bcastState.ID]
			if !existsInSystemState {
				delete(bcastSystem, bcastState.ID)
			}
			existingBcastState, existsInBcastSystem := bcastSystem[bcastState.ID]
			if existsInBcastSystem {
				if bcastState.SequenceNumber > existingBcastState.SequenceNumber {
					bcastSystem[bcastState.ID] = bcastState
					systemState[bcastState.ID] = bcastState.ElevState
				}
			} else {
				bcastSystem[bcastState.ID] = bcastState
				systemState[bcastState.ID] = bcastState.ElevState
			}
		}
	}
}

func StateBroadcaster(elevatorState <-chan Elev, systemState map[string]Elev, id string) {
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
	ticker := time.NewTicker(config.HEARTBEAT)
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
