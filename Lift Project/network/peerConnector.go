package network

import (
	"fmt"
	"mymodule/assigner"
	"mymodule/network/peers"
)

func PeerConnector(id string, world *assigner.World) {

	peerUpdateCh := make(chan peers.PeerUpdate)
	peerTxEnable := make(chan bool)

	go peers.Transmitter(15647, id, peerTxEnable)
	go peers.Receiver(15647, peerUpdateCh)
	go peerUpdater(peerUpdateCh, world)

}

func peerUpdater(peerUpdateCh chan peers.PeerUpdate, world *assigner.World) {
	for {
		select {
		case p := <-peerUpdateCh:
			fmt.Printf("Peer update:\n")
			fmt.Printf("  Peers:    %q\n", p.Peers)
			fmt.Printf("  New:      %q\n", p.New)
			fmt.Printf("  Lost:     %q\n", p.Lost)

			for _, element := range p.Lost {
				delete(world.Map, element)
				print("element was removed from world")
			}

		}
	}
}
