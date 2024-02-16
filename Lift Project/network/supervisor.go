package network

//
// import (
// 	"mymodule/assigner"
// 	. "mymodule/elevator"
// 	. "mymodule/network/peers"
// 	"time"
// )

// type Role int

// const (
// 	Worker Role = iota
// 	Supervisor
// 	BackupSupervisor
// )

// type ElevatorRole struct {
// 	Elevator Elev
// 	Role     Role
// }

// type PeerConnectedElevators struct {
// 	Map map[string]ElevatorRole
// }

// //Det må legges inn en if statement i peers.go som starter supervisor om den er den eneste i Peers
// //Deretter må PeerConnectedElevators sendes til BackupSupervisor kontinuerlig.

// func Supervise(p chan PeerUpdate, world *assigner.World) {
// 	peerWorld := PeerConnectedElevators{Map: make(map[string]ElevatorRole)}

// 	for {
// 		select {
// 		case id := <-p.New:
// 			RoleDistributor(peerWorld, world, id)
// 		case id := <-p.Lost:
// 			if peerWorld.Map[id].Role == Supervisor {
// 				for _, elevatorRole := range peerWorld.Map {
// 					if elevatorRole.Role == BackupSupervisor {
// 						elevatorRole.Role = Supervisor
// 					} else if elevatorRole.Role == Worker {
// 						elevatorRole.Role = BackupSupervisor
// 						break
// 					}
// 				}
// 			}
// 			delete(peerWorld.Map, id)
// 		}
// 	}
// 	//Alle ordre som blir akseptert sendes til supervisor
// 	//Supervisor følger med på at alle ordre blir utført
// }

// func RoleDistributor(peerWorld PeerConnectedElevators, world *assigner.World, id string) {
// 	for _, elevator := range world.Map {
// 		if len(peerWorld.Map) == 0 {
// 			peerWorld.Map[id] = ElevatorRole{elevator, Supervisor}
// 		} else if len(peerWorld.Map) == 1 {
// 			peerWorld.Map[id] = ElevatorRole{elevator, BackupSupervisor}
// 		} else {
// 			peerWorld.Map[id] = ElevatorRole{elevator, Worker}
// 		}
// 	}
// }

// func BackupSupervise(p chan PeerUpdate, world *assigner.World) {
// 	//Kommuniser med Supervisor
// 	supervisorComm := make(chan bool)
// 	peerWorld := PeerConnectedElevators{Map: make(map[string]ElevatorRole)}

// 	// Watchdog timer
// 	watchdog := time.NewTicker(time.Second * 5) // Adjust the duration as needed

// 	go func() {
// 		for {
// 			select {
// 			case <-supervisorComm:
// 				// Reset the watchdog timer when a message is received from the supervisor
// 				watchdog.Reset(time.Second * 5) // Adjust the duration as needed
// 			case <-watchdog.C:
// 				// The watchdog timer has expired, which means the supervisor has crashed
// 				// Start the Supervise function as a goroutine
// 				go Supervise(peerWorld, world)
// 				// Stop running the BackupSupervise function
// 				return
// 			}
// 		}
// 	}()
// }
