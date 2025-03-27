package worldView

import (
	"TTK4145-Heislab/configuration"
	"TTK4145-Heislab/singleElevator"
	"fmt"
)

func validateWorldView(wv WorldView) bool {
	if wv.ID == "" {
		fmt.Println("Validation failed: WorldView ID is empty")
		return false
	}
	if wv.ElevatorStatusList == nil || len(wv.ElevatorStatusList) == 0 {
		fmt.Println("Validation failed: ElevatorStatusList is nil or empty")
		return false
	}
	for id, elevStateMsg := range wv.ElevatorStatusList {
		if !validateElevatorStateMsg(id, elevStateMsg) {
			fmt.Printf("Validation failed: Invalid elevator state for %s\n", id)
			return false
		}
	}
	if len(wv.HallOrderStatus) == 0 {
		fmt.Println("Validation failed: HallOrderStatus is not initialized")
		return false
	}
	for floor := range wv.HallOrderStatus {
		for btn := range wv.HallOrderStatus[floor] {
			order := wv.HallOrderStatus[floor][btn]
			if !validateOrder(order) {
				fmt.Printf("Validation failed: Invalid hall order at floor %d, button %d\n", floor, btn)
				return false
			}
		}
	}
	return true
}

func validateElevatorStateMsg(id string, elevStateMsg ElevStateMessage) bool {
	if elevStateMsg.Cab == nil || len(elevStateMsg.Cab) != configuration.NumFloors {
		fmt.Printf("Validation failed: CabRequests not properly initialized for elevator %s\n", id)
		return false
	}
	for floor, order := range elevStateMsg.Cab {
		if !validateOrder(order) {
			fmt.Printf("Validation failed: Invalid cab order at floor %d for elevator %s\n", floor, id)
			return false
		}
	}
	if !validateElevatorState(elevStateMsg.Elev) {
		fmt.Printf("Validation failed: Invalid elevator core state for %s\n", id)
		return false
	}
	return true
}

func validateOrder(order configuration.OrderMessage) bool {
	if order.AckList == nil {
		fmt.Println("Validation failed: AckList is nil")
		return false
	}
	return true
}

func validateElevatorState(state singleElevator.Elevator) bool {
	if state.Floor < 0 || state.Floor >= configuration.NumFloors {
		fmt.Printf("Validation failed: Invalid floor value %d\n", state.Floor)
		return false
	}
	if state.Behaviour != singleElevator.Idle && state.Behaviour != singleElevator.Moving && state.Behaviour != singleElevator.DoorOpen {
		fmt.Printf("Validation failed: Invalid behavior %d\n", state.Behaviour)
		return false
	}
	if state.Behaviour == singleElevator.Moving && state.Behaviour == singleElevator.Idle {
		fmt.Println("Validation failed: Elevator cannot be both Moving and Idle at the same time")
		return false
	}
	return true
}
