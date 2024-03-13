package systemStateSynchronizer

import (
	"fmt"
	"mymodule/config"
	"mymodule/network/bcast"
	. "mymodule/types"
	"time"
)

func SystemStateSynchronizer(
	//addNode <-chan string,
	//removeNode <-chan string,
	requestSystemState <-chan bool,
	nodeLeft chan<- string,
	elevatorState <-chan Elev,
	systemState chan<- map[string]Elev,

) {

	broadcastStateRx := make(chan BcastState, 10)
	broadcastStateTx := make(chan BcastState, 10)
	updateFromBcast := make(chan map[string]Elev, 10)
	removeBcastNode := make(chan string, 10)
	heartBeat := make(chan HeartBeat, 256)
	heartBeatMissing := make(chan string, 10)

	go repeater(elevatorState, broadcastStateTx)
	go bcast.Transmitter(config.Broadcast_state_port, broadcastStateTx)
	go bcast.Receiver(config.Broadcast_state_port, broadcastStateRx)
	go updateBcastSystemState(updateFromBcast, broadcastStateRx, removeBcastNode, heartBeat)
	go checkHeartbeats(heartBeat, heartBeatMissing)

	localSystemState := make(map[string]Elev)
	for {
		select {
		// case id := <-addNode:
		// 	localSystemState[id] = Elev{}

		// case id := <-removeNode:
		// 	delete(localSystemState, id)
		// 	removeBcastNode <- id

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
				bcastSystem[bcastState.ID] = bcastState //node entered
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
		//fmt.Printf("Received heartbeat from %s\n", hb.ID)
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
		//time.Sleep(1 * time.Second)
	}
}

func repeater(elevatorState <-chan Elev, broadcastStateTx chan<- BcastState) {
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
				ID:             config.Self_id,
				SequenceNumber: i,
			}
			broadcastStateTx <- broadcastState
		}
	}
}
