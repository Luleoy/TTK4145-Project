package main

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
