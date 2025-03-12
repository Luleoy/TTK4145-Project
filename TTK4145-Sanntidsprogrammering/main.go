package main

import (
	"TTK4145-Heislab/Network-go/network/bcast"
	"TTK4145-Heislab/Network-go/network/peers"
	"TTK4145-Heislab/communication"
	"TTK4145-Heislab/configuration"
	"TTK4145-Heislab/driver-go/elevio"
	"TTK4145-Heislab/single_elevator"
	"TTK4145-Heislab/worldview"
	"flag"
	"fmt"
)

func main() {
	fmt.Println("Elevator System Starting...")

	idflag := flag.String("id", "default_id", "Id of this peer")
	elevioPortFlag := flag.String("ePort", "15657", "Port for elevio")
	flag.Parse()
	elevatorID := *idflag
	fmt.Println("My id: ", elevatorID)
	fmt.Println("Elevio port: ", *elevioPortFlag)

	// Initialize elevator hardware
	elevio.Init("localhost:"+*elevioPortFlag, configuration.NumFloors) //elevator - har laget ports i configuration

	// Communication channels
	newOrderChannel := make(chan single_elevator.Orders, configuration.Buffer)
	completedOrderChannel := make(chan elevio.ButtonEvent, configuration.Buffer)
	buttonPressedChannel := make(chan elevio.ButtonEvent)
	WorldViewTXChannel := make(chan worldview.WorldView)
	WorldViewRXChannel := make(chan worldview.WorldView)
	IDPeersChannel := make(chan []string)
	peerUpdateChannel := make(chan peers.PeerUpdate)

	go bcast.Transmitter(configuration.BroadcastPort, WorldViewTXChannel)
	go bcast.Receiver(configuration.BroadcastPort, WorldViewRXChannel)

	enableTransmit := make(chan bool)
	go peers.Transmitter(configuration.PeersPort, elevatorID, enableTransmit) //vi sender aldri noe inn i peers: transmitEnable <-chan bool
	go peers.Receiver(configuration.PeersPort, peerUpdateChannel)

	// Start FSM
	go elevio.PollButtons(buttonPressedChannel)
	//har started polling på obstruction, floorsensor, stopbutton i FSM

	// go single_elevator.OrderManager(newOrderChannel, completedOrderChannel, buttonPressedChannel)
	//go order_manager.Run(newOrderChannel, completedOrderChannel, buttonPressedChannel, network_tx, network_rx) - order manager erstattes
	go single_elevator.SingleElevator(newOrderChannel, completedOrderChannel)
	go communication.CommunicationHandler(elevatorID, peerUpdateChannel, IDPeersChannel)
	go worldview.WorldViewManager(elevatorID, WorldViewTXChannel, WorldViewRXChannel, buttonPressedChannel, newOrderChannel, completedOrderChannel, IDPeersChannel)

	select {}
}

/*
Network module
- UDP connection (packet loss) - packet sending and receiving (message format - JSON?) **concurrency
- Broadcasting (peer addresses, goroutine to periodically broadcast the elevator's state to all other peers)
- Message handling (message serialization/deserialization)

Peer to Peer module
- peer discovery
- message exchange
- peer failures
- synchronize the states

Assigner/Decision making module (cost function)
Fault Tolerance
*/
