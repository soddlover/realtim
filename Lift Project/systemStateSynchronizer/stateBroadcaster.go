package systemStateSynchronizer

// import (
// 	"fmt"
// 	"mymodule/config"
// 	"mymodule/network/bcast"
// 	. "mymodule/types"
// 	"time"
// // )

// var lastHeartbeat = make(map[string]time.Time)

// func CheckHeartbeats(lostConn chan string) {
// 	lastReported := make(map[string]bool)
// 	for {
// 		time.Sleep(1 * time.Second)

// 		for id, last := range lastHeartbeat {
// 			if time.Since(last) > config.HEARTBEAT_DEADLINE {

// 				if !lastReported[id] {
// 					fmt.Printf("No heartbeat from %s for more than 3 seconds\n", id)
// 					lostConn <- id
// 					lastReported[id] = true
// 				}
// 			} else {
// 				lastReported[id] = false
// 			}
// 		}
// 	}
// }

// func updateBroadcastworld(systemState map[string]Elev, broadcastStateRx <-chan BcastState) {
// 	bcastSystem := make(map[string]BcastState)
// 	for {
// 		bcastState := <-broadcastStateRx
// 		lastHeartbeat[bcastState.ID] = time.Now()
// 		_, existsInSystemState := systemState[bcastState.ID]
// 		if !existsInSystemState {
// 			delete(bcastSystem, bcastState.ID)
// 		}
// 		existingBcastState, existsInBcastSystem := bcastSystem[bcastState.ID]
// 		if existsInBcastSystem {
// 			if bcastState.SequenceNumber > existingBcastState.SequenceNumber {
// 				bcastSystem[bcastState.ID] = bcastState
// 				systemState[bcastState.ID] = bcastState.ElevState
// 			}
// 		} else {
// 			bcastSystem[bcastState.ID] = bcastState
// 			systemState[bcastState.ID] = bcastState.ElevState
// 		}
// 	}
// }

// func StateBroadcaster(elevatorState <-chan Elev, systemState chan map[string]Elev, id string) {

// }
