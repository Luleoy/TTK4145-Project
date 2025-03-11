/* package main

import (
	"TTK4145-Heislab/configuration"
	"TTK4145-Heislab/driver-go/elevio"
	"TTK4145-Heislab/single_elevator"
	"fmt"
)

func main() {
	fmt.Println("Elevator System Starting...")

	// Initialize elevator hardware
	numFloors := configuration.NumFloors
	elevio.Init("localhost:15658", numFloors) //elevator

	// Communication channels
	newOrderChannel := make(chan single_elevator.Orders, configuration.Buffer)
	completedOrderChannel := make(chan elevio.ButtonEvent, configuration.Buffer)
	newLocalStateChannel := make(chan single_elevator.Elevator, configuration.Buffer)
	buttonPressedChannel := make(chan elevio.ButtonEvent)

	// Polling channels
	// drv_buttons := make(chan elevio.ButtonEvent)
	// drv_floors := make(chan int)
	// drv_obstr := make(chan bool)
	// drv_stop := make(chan bool)

	// Start FSM
	go elevio.PollButtons(buttonPressedChannel)

	// go single_elevator.OrderManager(newOrderChannel, completedOrderChannel, buttonPressedChannel)
	//go order_manager.Run(newOrderChannel, completedOrderChannel, buttonPressedChannel, network_tx, network_rx) - order manager erstattes
	go single_elevator.SingleElevator(newOrderChannel, completedOrderChannel, newLocalStateChannel)

	// time.Sleep(10*time.Second)
	// exampleOrder := single_elevator.Orders {}
	// exampleOrder[0][1] = true

	// newOrderChannel <- exampleOrder

	//go order manager

	// Start polling inputs
	// go elevio.PollButtons(drv_buttons)
	// go elevio.PollFloorSensor(drv_floors) gjort
	// go elevio.PollObstructionSwitch(drv_obstr) gjort
	// go elevio.PollStopButton(drv_stop) gjort

	select {}
}


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

package main

import (
	"fmt"
	"TTK4145-Heislab/worldview"
	"TTK4145-Heislab/configuration"
)

func TestMergeWorldViews() {
	// Opprett et lokalt verdensbilde (localWorldView)
	elevatorID := "elev1"
	localWorldView := worldview.InitializeWorldView(elevatorID)
	localWorldView.HallOrderStatus[1][0].StateofOrder = configuration.UnConfirmed

	// Sett opp en annen versjon av verdensbildet (updatedWorldView)
	updatedWorldView := worldview.InitializeWorldView(elevatorID)

	// Definer aktive heiser
	IDsAliveElevators := []string{"elev1", "elev2"}

	// Simuler at en heis trykker på en knapp i `updatedWorldView`
	updatedWorldView.HallOrderStatus[1][0].StateofOrder = configuration.UnConfirmed
	updatedWorldView.HallOrderStatus[1][0].AckList["elev2"] = true

	// Simuler at en annen heis har en cab-bestilling
	updatedWorldView.ElevatorStatusList["elev2"] = worldview.ElevStateMsg{
		Cab: []configuration.OrderMsg{
			{StateofOrder: configuration.Confirmed, AckList: map[string]bool{"elev2": true}},
			{StateofOrder: configuration.UnConfirmed, AckList: map[string]bool{"elev2": true}},
			{StateofOrder: configuration.None},
			{StateofOrder: configuration.None},
		},
	}

	// Kall funksjonen vi tester
	mergedWorldView := worldview.MergeWorldViews(localWorldView, updatedWorldView, IDsAliveElevators)
	
	// Forventet output: HallOrderStatus[1][0] skal bli Confirmed hvis alle aktive heiser har ACK-et
	expectedHallStatus := configuration.Confirmed
	if mergedWorldView.HallOrderStatus[1][0].StateofOrder != expectedHallStatus {
		fmt.Printf("Feil: HallOrderStatus[1][0] er %v, forventet %v\n",
			mergedWorldView.HallOrderStatus[1][0].StateofOrder, expectedHallStatus)
	}

	// Sjekk om ACK-listene er riktig oppdatert
	if !mergedWorldView.HallOrderStatus[1][0].AckList["elev1"] || !mergedWorldView.HallOrderStatus[1][0].AckList["elev2"] {
		fmt.Println("Feil: ACK-listen inneholder ikke alle aktive heiser")
	}

	// Sjekk om cab-bestillingene er korrekt slått sammen
	expectedCabStatus := configuration.Confirmed
	if mergedWorldView.ElevatorStatusList["elev2"].Cab[0].StateofOrder != expectedCabStatus {
		fmt.Printf("Feil: Elev2 Cab[0] er %v, forventet %v\n",
			mergedWorldView.ElevatorStatusList["elev2"].Cab[0].StateofOrder, expectedCabStatus)
	}

	// Utskrift for vellykket test
	fmt.Println("Test MergeWorldViews fullført!")
}

func main() {
	TestMergeWorldViews()
}


