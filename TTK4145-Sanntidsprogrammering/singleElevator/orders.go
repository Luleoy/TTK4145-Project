package singleElevator

import (
	"TTK4145-Heislab/configuration"
	"TTK4145-Heislab/driver-go/elevio"
	"time"
)

type Orders [configuration.NumFloors][configuration.NumButtons]bool

type DirectionBehaviourPair struct {
	Direction elevio.MotorDirection
	Behaviour Behaviour
}

func hasOrderAtFloor(orders Orders, floor int) bool {
	for button := 0; button < configuration.NumButtons; button++ {
		if orders[floor][button] {
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
		if orders[floor][elevio.BT_HallDown] && !hasOrdersAbove(orders, floor) {
			return true
		}
	case Down:
		if orders[floor][elevio.BT_HallDown] {
			return true
		}
		if orders[floor][elevio.BT_HallUp] && !hasOrdersBelow(orders, floor) {
			return true
		}
	case Stop:
		panic("direction should not be stop")
	}
	return false

}

func hasOrdersAbove(orders Orders, floor int) bool {
	for f := floor + 1; f < configuration.NumFloors; f++ {
		if hasOrderAtFloor(orders, f) {
			return true
		}
	}
	return false
}

func hasOrdersBelow(orders Orders, floor int) bool {
	for f := floor - 1; f >= 0; f-- {
		if hasOrderAtFloor(orders, f) {
			return true
		}
	}
	return false
}

func orderCompletedatCurrentFloor(floor int, direction Direction, completedOrderChannel chan<- elevio.ButtonEvent, OrderMatrix Orders) {
	if OrderMatrix[floor][2] {
		completedOrderChannel <- elevio.ButtonEvent{Floor: floor, Button: elevio.BT_Cab}
	}
	switch direction {
	case Direction(elevio.MD_Up):
		if OrderMatrix[floor][elevio.BT_HallUp] {
			completedOrderChannel <- elevio.ButtonEvent{Floor: floor, Button: elevio.BT_HallUp}
		} else if OrderMatrix[floor][elevio.BT_HallDown] && !hasOrdersAbove(OrderMatrix, floor) {
			completedOrderChannel <- elevio.ButtonEvent{Floor: floor, Button: elevio.BT_HallDown}
		}
	case Direction(elevio.MD_Down):
		if OrderMatrix[floor][elevio.BT_HallDown] {
			completedOrderChannel <- elevio.ButtonEvent{Floor: floor, Button: elevio.BT_HallDown}
		} else if OrderMatrix[floor][elevio.BT_HallUp] && !hasOrdersBelow(OrderMatrix, floor) {
			completedOrderChannel <- elevio.ButtonEvent{Floor: floor, Button: elevio.BT_HallUp}
		}
	case Direction(elevio.MD_Stop):
		if !hasOrdersAbove(OrderMatrix, floor) && !hasOrdersBelow(OrderMatrix, floor) {
			if OrderMatrix[floor][elevio.BT_HallUp] {
				completedOrderChannel <- elevio.ButtonEvent{Floor: floor, Button: elevio.BT_HallUp}
			}
			if OrderMatrix[floor][elevio.BT_HallDown] {
				completedOrderChannel <- elevio.ButtonEvent{Floor: floor, Button: elevio.BT_HallDown}
			}
		} else {
			if hasOrdersAbove(OrderMatrix, floor) && OrderMatrix[floor][elevio.BT_HallUp] {
				completedOrderChannel <- elevio.ButtonEvent{Floor: floor, Button: elevio.BT_HallUp}
			}
			if hasOrdersBelow(OrderMatrix, floor) && OrderMatrix[floor][elevio.BT_HallDown] {
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

func ordersChooseDirection(floor int, direction Direction, OrderMatrix Orders) DirectionBehaviourPair {
	switch direction {
	case Up:
		if hasOrdersAbove(OrderMatrix, floor) {
			return DirectionBehaviourPair{elevio.MD_Up, Moving}
		} else if hasOrderAtFloor(OrderMatrix, floor) {
			return DirectionBehaviourPair{elevio.MD_Down, DoorOpen}
		} else if hasOrdersBelow(OrderMatrix, floor) {
			return DirectionBehaviourPair{elevio.MD_Down, Moving}
		} else {
			return DirectionBehaviourPair{elevio.MD_Stop, Idle}
		}
	case Down:
		if hasOrdersBelow(OrderMatrix, floor) {
			return DirectionBehaviourPair{elevio.MD_Down, Moving}
		} else if hasOrderAtFloor(OrderMatrix, floor) {
			return DirectionBehaviourPair{elevio.MD_Up, DoorOpen}
		} else if hasOrdersAbove(OrderMatrix, floor) {
			return DirectionBehaviourPair{elevio.MD_Up, Moving}
		} else {
			return DirectionBehaviourPair{elevio.MD_Stop, Idle}
		}
	case Stop:
		if hasOrderAtFloor(OrderMatrix, floor) {
			return DirectionBehaviourPair{elevio.MD_Stop, DoorOpen}
		} else if hasOrdersAbove(OrderMatrix, floor) {
			return DirectionBehaviourPair{elevio.MD_Up, Moving}
		} else if hasOrdersBelow(OrderMatrix, floor) {
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
		time.Sleep(100 * time.Millisecond)
	}
}
