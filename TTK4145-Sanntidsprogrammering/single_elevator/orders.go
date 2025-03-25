package single_elevator

import (
	"TTK4145-Heislab/configuration"
	"TTK4145-Heislab/driver-go/elevio"
	"fmt"
	"time"
)

/*
func SetLights(orderMatrix Orders) { //skru av og på lys
	for f := 0; f < configuration.NumFloors; f++ {
		for b := 0; b < configuration.NumButtons; b++ {
			elevio.SetButtonLamp(elevio.ButtonType(b), f, orderMatrix[f][b])
		}
	}
} */

type Orders [configuration.NumFloors][configuration.NumButtons]bool //creating matrix to take orders. floors*buttons

func orderHere(orders Orders, floor int) bool {
	for b := 0; b < configuration.NumButtons; b++ {
		if orders[floor][b] { // Hvis det finnes en aktiv forespørsel
			return true
		}
	}
	return false
}

func shouldStopAtFloor(orders Orders, floor int, direction Direction) bool {
	anyOrders := false
	for i := 0; i < configuration.NumFloors; i++ {
		for j := 0; j < configuration.NumButtons; j++ {
			anyOrders = anyOrders || orders[i][j]
		}
	}
	if !anyOrders {
		fmt.Println("Stopping due to no orders")
		return true
	}
	if orders[floor][elevio.BT_Cab] || floor == 0 || floor == configuration.NumFloors-1 {
		return true
	}
	switch direction {
	case Up:
		if orders[floor][elevio.BT_HallUp] {
			return true
		}
		if orders[floor][elevio.BT_HallDown] && !ordersAbove(orders, floor) {
			return true
		}
	case Down:
		if orders[floor][elevio.BT_HallDown] {
			return true
		}
		if orders[floor][elevio.BT_HallUp] && !ordersBelow(orders, floor) {
			return true
		}
	case Stop:
		panic("direction should not be stop")
	}
	return false

}

func ordersAbove(orders Orders, floor int) bool {
	for f := floor + 1; f < configuration.NumFloors; f++ {
		if orderHere(orders, f) {
			return true
		}
	}
	return false
}

func ordersBelow(orders Orders, floor int) bool {
	for f := floor - 1; f >= 0; f-- {
		if orderHere(orders, f) {
			return true
		}
	}
	return false
}

func OrderCompletedatCurrentFloor(floor int, direction Direction, completedOrderChannel chan<- elevio.ButtonEvent, OrderMatrix Orders) {
	if OrderMatrix[floor][2] {
		completedOrderChannel <- elevio.ButtonEvent{Floor: floor, Button: elevio.BT_Cab}
	}
	switch direction {
	case Direction(elevio.MD_Up):
		if OrderMatrix[floor][elevio.BT_HallUp] {
			completedOrderChannel <- elevio.ButtonEvent{Floor: floor, Button: elevio.BT_HallUp}
		} else if OrderMatrix[floor][elevio.BT_HallDown] && !ordersAbove(OrderMatrix, floor) {
			completedOrderChannel <- elevio.ButtonEvent{Floor: floor, Button: elevio.BT_HallDown}
		}
	case Direction(elevio.MD_Down):
		if OrderMatrix[floor][elevio.BT_HallDown] {
			completedOrderChannel <- elevio.ButtonEvent{Floor: floor, Button: elevio.BT_HallDown}
		} else if OrderMatrix[floor][elevio.BT_HallUp] && !ordersBelow(OrderMatrix, floor) {
			completedOrderChannel <- elevio.ButtonEvent{Floor: floor, Button: elevio.BT_HallUp}
		}
	case Direction(elevio.MD_Stop):
		if !ordersAbove(OrderMatrix, floor) && !ordersBelow(OrderMatrix, floor) {
			if OrderMatrix[floor][elevio.BT_HallUp] {
				completedOrderChannel <- elevio.ButtonEvent{Floor: floor, Button: elevio.BT_HallUp}
			}
			if OrderMatrix[floor][elevio.BT_HallDown] {
				completedOrderChannel <- elevio.ButtonEvent{Floor: floor, Button: elevio.BT_HallDown}
			}
		} else {
			// Hvis det er bestillinger over eller under, fjern kun bestillingen som er i samsvar med retningen
			if ordersAbove(OrderMatrix, floor) && OrderMatrix[floor][elevio.BT_HallUp] {
				completedOrderChannel <- elevio.ButtonEvent{Floor: floor, Button: elevio.BT_HallUp}
			}
			if ordersBelow(OrderMatrix, floor) && OrderMatrix[floor][elevio.BT_HallDown] {
				completedOrderChannel <- elevio.ButtonEvent{Floor: floor, Button: elevio.BT_HallDown}
			}
		}
	}
}

/*
order manager for single elevator
func OrderManager(newOrderChannel chan<- Orders,
	completedOrderChannel <-chan elevio.ButtonEvent, //sende-kanal
	buttonPressedChannel <-chan elevio.ButtonEvent) { //kun lesing av kanal
	OrderMatrix := [configuration.NumFloors][configuration.NumButtons]bool{}
	for {
		select {
		case buttonPressed := <-buttonPressedChannel:
			OrderMatrix[buttonPressed.Floor][buttonPressed.Button] = true
			SetLights(OrderMatrix)
			newOrderChannel <- OrderMatrix
		case ordercompletedbyfsm := <-completedOrderChannel:
			OrderMatrix[ordercompletedbyfsm.Floor][ordercompletedbyfsm.Button] = false
			SetLights(OrderMatrix)
			newOrderChannel <- OrderMatrix
		}
	}
}*/

//output fra hallrequest assigner som skal sendes inn i ordermanager
//vi har enere allerede. er ikke "nye orders" men heller orders in general
//hall request assigner skal kjøres kontinuerlig

type DirectionBehaviourPair struct {
	Direction elevio.MotorDirection
	Behaviour Behaviour //vi skal hente ut Behaviour (moving, idle, dooropen)
}

func ordersChooseDirection(floor int, direction Direction, OrderMatrix Orders) DirectionBehaviourPair {
	switch direction {
	case Up:
		if ordersAbove(OrderMatrix, floor) {
			return DirectionBehaviourPair{elevio.MD_Up, Moving}
		} else if orderHere(OrderMatrix, floor) {
			return DirectionBehaviourPair{elevio.MD_Down, DoorOpen}
		} else if ordersBelow(OrderMatrix, floor) {
			return DirectionBehaviourPair{elevio.MD_Down, Moving}
		} else {
			return DirectionBehaviourPair{elevio.MD_Stop, Idle}
		}
	case Down:
		if ordersBelow(OrderMatrix, floor) {
			return DirectionBehaviourPair{elevio.MD_Down, Moving}
		} else if orderHere(OrderMatrix, floor) {
			return DirectionBehaviourPair{elevio.MD_Up, DoorOpen}
		} else if ordersAbove(OrderMatrix, floor) {
			return DirectionBehaviourPair{elevio.MD_Up, Moving}
		} else {
			return DirectionBehaviourPair{elevio.MD_Stop, Idle}
		}
	case Stop:
		if orderHere(OrderMatrix, floor) {
			return DirectionBehaviourPair{elevio.MD_Stop, DoorOpen}
		} else if ordersAbove(OrderMatrix, floor) {
			return DirectionBehaviourPair{elevio.MD_Up, Moving}
		} else if ordersBelow(OrderMatrix, floor) {
			return DirectionBehaviourPair{elevio.MD_Down, Moving}
		} else {
			return DirectionBehaviourPair{elevio.MD_Stop, Idle}
		}
	default:
		return DirectionBehaviourPair{elevio.MD_Stop, Idle}
	}
}

func findClosestFloor() int {
	for {
		floor := elevio.GetFloor()
		if floor != -1 {
			return floor
		}
		time.Sleep(100 * time.Millisecond) // Sjekker hvert 100 ms
	}
}
