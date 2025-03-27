package peerTracker

import (
	"TTK4145-Heislab/Network-go/network/peers"
	"fmt"
)

func TrackActivePeers(
	elevatorID string,
	peerUpdateChannel <-chan peers.PeerUpdate,
	IDPeersChannel chan<- []string,
) {
	for peers := range peerUpdateChannel {
		fmt.Printf("Peer update:\n")
		fmt.Printf("  Peers:    %q\n", peers.Peers)
		fmt.Printf("  New:      %q\n", peers.New)
		fmt.Printf("  Lost:     %q\n", peers.Lost)
		IDPeersChannel <- peers.Peers
	}
}
