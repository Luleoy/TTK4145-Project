package main

import (
	"TTK4145-Heislab/Network-go/network/bcast"
	"TTK4145-Heislab/Network-go/network/peers"
	"TTK4145-Heislab/configuration"
	"TTK4145-Heislab/driver-go/elevio"
	"TTK4145-Heislab/peerTracker"
	"TTK4145-Heislab/singleElevator"
	"TTK4145-Heislab/worldView"
	"flag"
	"time"
)

func main() {

	idflag := flag.String("id", "default_id", "Id of this peer")
	elevioPortFlag := flag.String("ePort", "15657", "Port for elevio")
	flag.Parse()
	elevatorID := *idflag
	elevio.Init("localhost:"+*elevioPortFlag, configuration.NumFloors)

	newOrderChannel := make(chan singleElevator.Orders, configuration.Buffer)
	completedOrderChannel := make(chan elevio.ButtonEvent, configuration.Buffer)
	buttonPressedChannel := make(chan elevio.ButtonEvent, configuration.Buffer)
	WorldViewTXChannel := make(chan worldView.WorldView, configuration.Buffer)
	WorldViewRXChannel := make(chan worldView.WorldView, configuration.Buffer)
	IDPeersChannel := make(chan []string)
	peerUpdateChannel := make(chan peers.PeerUpdate)
	elevatorStateChannel := make(chan singleElevator.Elevator, configuration.Buffer)
	elevatorTimeoutTimer := time.NewTimer(5 * time.Second)

	go bcast.Transmitter(configuration.BroadcastPort, WorldViewTXChannel)
	go bcast.Receiver(configuration.BroadcastPort, WorldViewRXChannel)

	enableTransmit := make(chan bool)
	go peers.Transmitter(configuration.PeersPort, elevatorID, enableTransmit)
	go peers.Receiver(configuration.PeersPort, peerUpdateChannel)

	go elevio.PollButtons(buttonPressedChannel)
	go singleElevator.SingleElevatorFsm(newOrderChannel, completedOrderChannel, elevatorStateChannel)
	go peerTracker.TrackActivePeers(elevatorID, peerUpdateChannel, IDPeersChannel)
	go worldView.WorldViewManager(elevatorID, WorldViewTXChannel, WorldViewRXChannel, buttonPressedChannel, newOrderChannel, completedOrderChannel, IDPeersChannel, elevatorStateChannel, elevatorTimeoutTimer)

	select {}
}
