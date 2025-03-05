package single_elevator

import (
	"TTK4145-Heislab/configuration"
	"TTK4145-Heislab/driver-go/elevio"
)

func SetLights(orderMatrix Orders) { //skru av og på lys
	for f := 0; f < configuration.NumFloors; f++ {
		for b := 0; b < configuration.NumButtons; b++ {
			elevio.SetButtonLamp(elevio.ButtonType(b), f, orderMatrix[f][b])
		}
	}
}

type Orders [configuration.NumFloors][configuration.NumButtons]bool //creating matrix to take orders. floors*buttons

func orderHere(orders Orders, floor int) bool {
	for b := 0; b < configuration.NumButtons; b++ {
		if orders[floor][b] { // Hvis det finnes en aktiv forespørsel
			return true
		}
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

func OrderCompletedatCurrentFloor(floor int, direction Direction, completedOrderChannel chan<- elevio.ButtonEvent) {
	completedOrderChannel <- elevio.ButtonEvent{Floor: floor, Button: elevio.BT_Cab}
	switch direction {
	case Direction(elevio.MD_Up):
		completedOrderChannel <- elevio.ButtonEvent{Floor: floor, Button: elevio.BT_HallUp}
	case Direction(elevio.MD_Down):
		completedOrderChannel <- elevio.ButtonEvent{Floor: floor, Button: elevio.BT_HallDown}
	case Direction(elevio.MD_Stop):
		completedOrderChannel <- elevio.ButtonEvent{Floor: floor, Button: elevio.BT_HallUp}
		completedOrderChannel <- elevio.ButtonEvent{Floor: floor, Button: elevio.BT_HallDown}
	}
}

/*
order manager for single elevator
func OrderManager(newOrderChannel chan<- Orders,
	completedOrderChannel <-chan elevio.ButtonEvent, //sende-kanal
	//newLocalStateChannel <-chan State, //sende-kanal - NÅR SKAL DENNE BRUKES?
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
