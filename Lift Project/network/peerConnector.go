package network

import (
	"mymodule/config"
	"mymodule/network/peers"
	. "mymodule/types"
)

func PeerConnector(id string, world *World, channels Channels) {

	peerUpdateCh := make(chan peers.PeerUpdate)
	peerTxEnable := make(chan bool)

	println("PeerConnector started, transmitting id: ", id)
	go peers.Transmitter(config.Peer_port, id, peerTxEnable)
	go peers.Receiver(config.Peer_port, peerUpdateCh)
	//on startup wait for connections then check if only one is online

	//OutgoingOrder := make(chan Order)

	//This code is just to higlight which channels are available

}
