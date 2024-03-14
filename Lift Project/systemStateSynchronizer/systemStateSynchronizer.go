package systemStateSynchronizer

import (
	"fmt"
	"mymodule/config"
	"mymodule/network/bcast"
	. "mymodule/types"
	"time"
)

func SystemStateSynchronizer(
	requestSystemState <-chan bool,
	nodeLeft chan<- string,
	elevatorState <-chan Elev,
	systemState chan<- map[string]Elev,
) {
	broadcastStateTx := make(chan BcastState, config.ELEVATOR_BUFFER_SIZE)
	go repeater(elevatorState, broadcastStateTx)
	go bcast.Transmitter(config.Broadcast_state_port, broadcastStateTx)
	broadcastStateRx := make(chan BcastState, config.ELEVATOR_BUFFER_SIZE)
	go bcast.Receiver(config.Broadcast_state_port, broadcastStateRx)
	updateFromBcast := make(chan map[string]Elev, config.ELEVATOR_BUFFER_SIZE)
	removeBcastNode := make(chan string, config.ELEVATOR_BUFFER_SIZE)
	heartBeat := make(chan HeartBeat, config.ELEVATOR_BUFFER_SIZE)
	go updateBcastSystemState(updateFromBcast, broadcastStateRx, removeBcastNode, heartBeat)
	heartBeatMissing := make(chan string, config.ELEVATOR_BUFFER_SIZE)
	go checkHeartbeats(heartBeat, heartBeatMissing)

	localSystemState := make(map[string]Elev)
	for {
		select {
		case id := <-heartBeatMissing:
			removeBcastNode <- id
			nodeLeft <- id

		case <-requestSystemState:
			systemState <- localSystemState

		case newSystemState := <-updateFromBcast:
			localSystemState = newSystemState
		}
	}
}

func updateBcastSystemState(
	updateFromBcast chan<- map[string]Elev,
	broadcastStateRx <-chan BcastState,
	removeNode <-chan string,
	heartBeat chan<- HeartBeat,
) {
	bcastSystem := make(map[string]BcastState)
	for {
		select {
		case bcastState := <-broadcastStateRx:
			heartBeat <- HeartBeat{ID: bcastState.ID, Time: time.Now()}
			currentBcastState, existsInBcastSystem := bcastSystem[bcastState.ID]
			if existsInBcastSystem && bcastState.SequenceNumber > currentBcastState.SequenceNumber {
				bcastSystem[bcastState.ID] = bcastState
			} else {
				bcastSystem[bcastState.ID] = bcastState
			}
			updateFromBcast <- convertToSystemState(bcastSystem)

		case id := <-removeNode:
			delete(bcastSystem, id)
			updateFromBcast <- convertToSystemState(bcastSystem)
		}
	}
}

func convertToSystemState(bcastSystem map[string]BcastState) map[string]Elev {
	systemState := make(map[string]Elev)
	for id, bcastState := range bcastSystem {
		systemState[id] = bcastState.ElevState
	}
	return systemState
}

func checkHeartbeats(heartBeat <-chan HeartBeat, lostConn chan<- string) {
	lastHeartBeat := make(map[string]time.Time)
	lastReported := make(map[string]bool)
	for {
		hb := <-heartBeat
		lastHeartBeat[hb.ID] = hb.Time

		for id, last := range lastHeartBeat {
			if time.Since(last) > config.HEARTBEAT_DEADLINE {
				if !lastReported[id] {
					fmt.Printf("No heartbeat from %s for more than 3 seconds\n", id)
					lostConn <- id
					lastReported[id] = true
				}
			} else {
				lastReported[id] = false
			}
		}
	}
}

func repeater(elevatorState <-chan Elev, broadcastStateTx chan<- BcastState) {
	ticker := time.NewTicker(config.HEARTBEAT)
	var broadcastState BcastState
	var lastElev Elev
	for i := 0; ; i++ {
		select {
		case elev := <-elevatorState:
			lastElev = elev
		case <-ticker.C:
			broadcastState = BcastState{
				ElevState:      lastElev,
				ID:             config.Self_id,
				SequenceNumber: i,
			}
			broadcastStateTx <- broadcastState
		}
	}
}
