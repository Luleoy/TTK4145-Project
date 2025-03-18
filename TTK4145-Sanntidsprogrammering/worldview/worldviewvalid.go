package worldview

import (
	"TTK4145-Heislab/configuration"
	"TTK4145-Heislab/single_elevator"
	"fmt"
)

// ValidateWorldView checks if the given WorldView instance is valid and returns a bool.
func ValidateWorldView(wv WorldView) bool {
	// 1. Check if ID is valid
	if wv.ID == "" {
		fmt.Println("Validation failed: WorldView ID is empty")
		return false
	}

	// 2. Check if ElevatorStatusList is initialized and not empty
	if wv.ElevatorStatusList == nil || len(wv.ElevatorStatusList) == 0 {
		fmt.Println("Validation failed: ElevatorStatusList is nil or empty")
		return false
	}

	// 3. Validate each elevator state
	for id, elevStateMsg := range wv.ElevatorStatusList {
		if !ValidateElevatorState(id, elevStateMsg) {
			fmt.Printf("Validation failed: Invalid elevator state for %s\n", id)
			return false
		}
	}

	// 4. Check if HallOrderStatus is properly initialized
	if len(wv.HallOrderStatus) == 0 {
		fmt.Println("Validation failed: HallOrderStatus is not initialized")
		return false
	}

	for floor := range wv.HallOrderStatus {
		for btn := range wv.HallOrderStatus[floor] {
			order := wv.HallOrderStatus[floor][btn]
			if !ValidateOrder(order) {
				fmt.Printf("Validation failed: Invalid hall order at floor %d, button %d\n", floor, btn)
				return false
			}
		}
	}

	// Everything is valid
	return true
}

func ValidateElevatorState(id string, elevStateMsg ElevStateMsg) bool {
	// Check if Cab array is properly initialized
	if elevStateMsg.Cab == nil || len(elevStateMsg.Cab) != configuration.NumFloors {
		fmt.Printf("Validation failed: CabRequests not properly initialized for elevator %s\n", id)
		return false
	}

	// Validate each Cab order
	for floor, order := range elevStateMsg.Cab {
		if !ValidateOrder(order) {
			fmt.Printf("Validation failed: Invalid cab order at floor %d for elevator %s\n", floor, id)
			return false
		}
	}

	// Ensure the elevator’s state is valid
	if !ValidateElevatorCoreState(elevStateMsg.Elev) {
		fmt.Printf("Validation failed: Invalid elevator core state for %s\n", id)
		return false
	}

	return true
}

func ValidateOrder(order configuration.OrderMsg) bool {
	if order.AckList == nil {
		fmt.Println("Validation failed: Order AckList is nil")
		return false
	}

	return true
}

func ValidateElevatorCoreState(state single_elevator.Elevator) bool {
	// Ensure floor is within a valid range
	if state.Floor < 0 || state.Floor >= configuration.NumFloors {
		fmt.Printf("Validation failed: Invalid floor value %d\n", state.Floor)
		return false
	}

	// Ensure behavior is valid
	if state.Behaviour != single_elevator.Idle && state.Behaviour != single_elevator.Moving && state.Behaviour != single_elevator.DoorOpen {
		fmt.Printf("Validation failed: Invalid behavior %d\n", state.Behaviour)
		return false
	}

	// Ensure an elevator cannot be both Moving and Idle
	if state.Behaviour == single_elevator.Moving && state.Behaviour == single_elevator.Idle {
		fmt.Println("Validation failed: Elevator cannot be both Moving and Idle at the same time")
		return false
	}

	return true
}
