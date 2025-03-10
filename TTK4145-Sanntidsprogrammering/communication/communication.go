package communication

import (
	"TTK4145-Heislab/Network-go/network/peers"
	"fmt"
)

func CommunicationHandler(

	elevatorID string,
	peerUpdateChannel <-chan peers.PeerUpdate,
	//peerTXEnableChannel chan<- bool,
	IDPeersChannel chan<- []string,

) {
	//fjernet select case fordi lytter kun på en kanal
	//oppdatere på hvilke heiser som er aktive ( når heiser kommer på og forsvinner fra nettverket)
	for peers := range peerUpdateChannel {
		fmt.Printf("Peer update:\n")
		fmt.Printf("  Peers:    %q\n", peers.Peers)
		fmt.Printf("  New:      %q\n", peers.New)
		fmt.Printf("  Lost:     %q\n", peers.Lost)

		// Oppdaterer aktive peers
		IDPeersChannel <- peers.Peers
	}
}
