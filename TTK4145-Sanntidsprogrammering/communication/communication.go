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
	NewlocalElevatorChannel <-chan single_elevator.State,
	peerTXEnableChannel chan<- bool,
	IDPeersChannel chan <- []string,

) {

	//initialisering
	localWorldView := worldview.InitializeWorldView(elevatorID)
	//numPeers := 0

	for {

		select {

		//case_ 5: Oppdateringer for den lokale heisen, trenger vi den??
		case newLocalElevator := <-NewlocalElevatorChannel: //listning to channel
			localWorldView.ElevatorStatusList[elevatorID] = newLocalElevator
			
			//cabRequest := GetCabRequests(newLocalElevator) 
			// //cabRequest brukes ikke videre i koden - CAB må hentes ut av WORLDVIEW 

		//Case 6:
		//oppdatere på hvilke heiser som er aktive ( når heiser kommer på og forsvinner fra nettverket)
		case peers := <-peerUpdateChannel: //lisning to channel

			//writing out updated info

			fmt.Printf("Peer update:\n")
			fmt.Printf("  Peers:    %q\n", peers.Peers)
			fmt.Printf("  New:      %q\n", peers.New)
			fmt.Printf("  Lost:     %q\n", peers.Lost)

			//Oppdaterer aktive peers
			//numPeers = len(peers.Peers)
			IDPeersChannel <- peers.Peers

			/*
			//finer om tapt heis utilgjengelig
			if localWorldView.ElevatorStatusList[peers.Lost[0]].Unavailable { //her må det gjøres noe
				worldview.AssignOrder(*&localWorldView, newOrderChannel) //har ikke assignedrequestschannel - VI BRUKER NEWORDERCHANNEL
				peerTXEnableChannel <- true
			} else {
				//tilgjengelig heis, fjernes tapt heis fra systemoversikt
				for i, ack := range localWorldView.Acklist {
					for _, lostPeer := range peers.Lost {
						delete(localWorldView.ElevatorStatusList, lostPeer)

						//Fjerner heisen fra Acklist
						if ack == lostPeer {
							localWorldView.Acklist = append(localWorldView.Acklist[:i], localWorldView.Acklist[i+1:]...)
						}
					}
				}
				//Redistribuer ordre
				worldview.AssignOrder(*&localWorldView, worldview.WorldViewManager())
			}
		}
			*/
	}
}
}