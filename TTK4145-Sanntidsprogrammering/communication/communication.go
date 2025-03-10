package communication

import (
	"TTK4145-Heislab/Network-go/network/peers"
	"TTK4145-Heislab/single_elevator"
	"TTK4145-Heislab/worldview"
	"fmt"
)

func CommunicationHandler(

	elevatorID string,
	peerUpdateChannel <-chan peers.PeerUpdate,
	NewlocalElevatorChannel <-chan single_elevator.Elevator,
	peerTXEnableChannel chan<- bool,
	IDPeersChannel chan<- []string,
	WorldViewRXChannel <-chan worldview.WorldView,

) {

	for {

		select {

		//oppdatere på hvilke heiser som er aktive ( når heiser kommer på og forsvinner fra nettverket)
		case peers := <-peerUpdateChannel: //lisning to channel

			//writing out updated info

			fmt.Printf("Peer update:\n")
			fmt.Printf("  Peers:    %q\n", peers.Peers)
			fmt.Printf("  New:      %q\n", peers.New)
			fmt.Printf("  Lost:     %q\n", peers.Lost)

			//Oppdaterer aktive peers
			IDPeersChannel <- peers.Peers
		}
	}
}
