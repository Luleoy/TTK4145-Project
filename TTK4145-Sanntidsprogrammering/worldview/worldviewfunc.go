package worldview

import (
	"TTK4145-Heislab/AssignerExecutable"
	"TTK4145-Heislab/configuration"
	"TTK4145-Heislab/driver-go/elevio"
	"TTK4145-Heislab/single_elevator"
)

func InitializeHallOrderStatus() [][configuration.NumButtons - 1]configuration.OrderMsg {
	HallOrderStatus := make([][configuration.NumButtons - 1]configuration.OrderMsg, configuration.NumFloors)
	for floor := range HallOrderStatus {
		for button := range HallOrderStatus[floor] {
			HallOrderStatus[floor][button] = configuration.OrderMsg{
				StateofOrder: configuration.None,
				AckList:      make(map[string]bool),
			}
		}
	}
	return HallOrderStatus
}

func InitializeCabOrders() []configuration.OrderMsg {
	CabOrders := make([]configuration.OrderMsg, configuration.NumFloors)
	for floor := range CabOrders {
		CabOrders[floor] = configuration.OrderMsg{
			StateofOrder: configuration.None,
			AckList:      make(map[string]bool),
		}
	}
	return CabOrders
}

func InitializeWorldView(elevatorID string) WorldView {
	wv := WorldView{
		ID:                 elevatorID,
		ElevatorStatusList: make(map[string]ElevStateMsg),
		HallOrderStatus:    InitializeHallOrderStatus(),
	}
	elevatorState := ElevStateMsg{
		Elev: single_elevator.Elevator{},
		Cab:  InitializeCabOrders(),
	}
	wv.ElevatorStatusList[elevatorID] = elevatorState
	return wv
}

func UpdateWorldViewWithButton(localWorldView *WorldView, buttonPressed elevio.ButtonEvent, isNewOrder bool) WorldView {
	if isNewOrder {
		switch buttonPressed.Button {
		case elevio.BT_HallUp, elevio.BT_HallDown:
			localWorldView.HallOrderStatus[buttonPressed.Floor][buttonPressed.Button] = configuration.OrderMsg{
				StateofOrder: configuration.UnConfirmed,
				AckList:      make(map[string]bool),
			}
			localWorldView.HallOrderStatus[buttonPressed.Floor][buttonPressed.Button].AckList[localWorldView.ID] = true
		case elevio.BT_Cab:
			localWorldView.ElevatorStatusList[localWorldView.ID].Cab[buttonPressed.Floor] = configuration.OrderMsg{
				StateofOrder: configuration.UnConfirmed,
				AckList:      make(map[string]bool),
			}
			localWorldView.ElevatorStatusList[localWorldView.ID].Cab[buttonPressed.Floor].AckList[localWorldView.ID] = true
		}
	} else {
		switch buttonPressed.Button {
		case elevio.BT_HallUp, elevio.BT_HallDown:
			localWorldView.HallOrderStatus[buttonPressed.Floor][buttonPressed.Button].StateofOrder = configuration.Completed
			ResetAckList(localWorldView)
		case elevio.BT_Cab:
			localWorldView.ElevatorStatusList[localWorldView.ID].Cab[buttonPressed.Floor].StateofOrder = configuration.Completed
			ResetAckList(localWorldView)
		}
	}
	return *localWorldView
}

func ResetAckList(localWorldView *WorldView) {
	for floor := range localWorldView.HallOrderStatus {
		for btn := range localWorldView.HallOrderStatus[floor] {
			localWorldView.HallOrderStatus[floor][btn].AckList = make(map[string]bool)
			localWorldView.HallOrderStatus[floor][btn].AckList[localWorldView.ID] = true
		}
	}
	for _, elevState := range localWorldView.ElevatorStatusList {
		for floor := range elevState.Cab {
			elevState.Cab[floor].AckList = make(map[string]bool)
			elevState.Cab[floor].AckList[localWorldView.ID] = true
		}
	}
}

func ConvertHallOrderStatestoBool(worldView WorldView) [][2]bool {
	boolMatrix := make([][2]bool, configuration.NumFloors)
	for floor := range boolMatrix {
		for button := 0; button < 2; button++ {
			if worldView.HallOrderStatus[floor][button].StateofOrder == configuration.Confirmed {
				boolMatrix[floor][button] = true
			} else {
				boolMatrix[floor][button] = false
			}
		}
	}
	return boolMatrix
}

func HRAInputFormatting(worldView WorldView, IDsAliveElevators []string) AssignerExecutable.HRAInput {
	hallRequests := ConvertHallOrderStatestoBool(worldView)
	elevatorStates := make(map[string]AssignerExecutable.HRAElevState)
	for _, elevatorID := range IDsAliveElevators {
		elevState, exists := worldView.ElevatorStatusList[elevatorID]
		if !exists {
			continue
		}
		if !elevState.Elev.Unavailable {
			cabRequests := make([]bool, configuration.NumFloors)
			for floor, cabOrder := range elevState.Cab {
				cabRequests[floor] = cabOrder.StateofOrder == configuration.Confirmed
			}
			elevatorStates[elevatorID] = AssignerExecutable.HRAElevState{
				Behaviour:   single_elevator.ToString(elevState.Elev.Behaviour),
				Floor:       elevState.Elev.Floor,
				Direction:   elevio.DirToString(elevio.MotorDirection(elevState.Elev.Direction)),
				CabRequests: cabRequests,
			}
		}
	}
	input := AssignerExecutable.HRAInput{
		HallRequests: hallRequests,
		States:       elevatorStates,
	}
	return input
}

func MergeCABandHRAout(OurHall [][2]bool, Ourcab []bool) single_elevator.Orders {
	var OrderMatrix single_elevator.Orders
	for floor, cabButton := range Ourcab {
		if cabButton {
			OrderMatrix[floor][2] = true
		}
	}
	for floor, buttons := range OurHall {
		for buttonType, isPressed := range buttons {
			if isPressed {
				OrderMatrix[floor][buttonType] = true
			}
		}
	}
	return OrderMatrix
}

func GetOurCAB(localWorldView WorldView, ourID string) []bool {
	cabOrders := localWorldView.ElevatorStatusList[ourID].Cab
	ourCab := make([]bool, len(cabOrders))
	for i, order := range cabOrders {
		ourCab[i] = order.StateofOrder == configuration.Confirmed
	}
	return ourCab
}

func SetLights(localWorldView WorldView) {
	for floor := range localWorldView.HallOrderStatus {
		for button := 0; button < 2; button++ {
			order := localWorldView.HallOrderStatus[floor][button]
			Light := (order.StateofOrder == configuration.Confirmed || order.StateofOrder == configuration.UnConfirmed)
			elevio.SetButtonLamp(elevio.ButtonType(button), floor, Light)
		}
	}
	for id, elevatorState := range localWorldView.ElevatorStatusList {
		if id == localWorldView.ID {
			for floor, order := range elevatorState.Cab {
				Light := (order.StateofOrder == configuration.Confirmed || order.StateofOrder == configuration.UnConfirmed)
				elevio.SetButtonLamp(elevio.BT_Cab, floor, Light)
			}
		}
	}
}

// HJELP
func AssignOrder(worldView WorldView, IDsAliveElevators []string) map[string][][2]bool {
	input := HRAInputFormatting(worldView, IDsAliveElevators)
	outputAssigner := AssignerExecutable.Assigner(input)
	return outputAssigner
}

// HJELP
func MergeWorldViews(localWorldView WorldView, updatedWorldView WorldView, IDsAliveElevators []string) WorldView {
	for id, state := range updatedWorldView.ElevatorStatusList { // Iterate over elevatorstatuslist in updatedWorldView and update the corresponding entries in the localWorldView
		localWorldView.ElevatorStatusList[id] = state
	}
	for floor := range localWorldView.HallOrderStatus { // Iterate over hall orders. Merge hallOrderStatus
		for button := range localWorldView.HallOrderStatus[floor] {
			// Get the local and updated orders for floor and button
			localOrder := &localWorldView.HallOrderStatus[floor][button]
			updatedOrder := updatedWorldView.HallOrderStatus[floor][button]
			if localOrder.AckList == nil {
				localOrder.AckList = make(map[string]bool)
			}
			for id := range updatedOrder.AckList {
				localOrder.AckList[id] = true
			}
			localOrder.AckList[localWorldView.ID] = true
			if localOrder.StateofOrder == configuration.UnConfirmed { //handle barrier condition
				allAcknowledged := true
				for _, id := range IDsAliveElevators { // Check if all alive elevators have acknowledged this order
					if !localOrder.AckList[id] {
						allAcknowledged = false
						break
					}
				}
				if allAcknowledged { // If all alive elevators have acknowledged, transition to CONFIRMED
					localOrder.StateofOrder = configuration.Confirmed
				}
			}
		}
	}
	for id, elevState := range localWorldView.ElevatorStatusList { // Iterate over cab orders. Merge cab orders and handle the barrier condition
		for floor := range elevState.Cab {
			localCabOrder := &localWorldView.ElevatorStatusList[id].Cab[floor]
			updatedCabOrder := updatedWorldView.ElevatorStatusList[id].Cab[floor]
			if localCabOrder.AckList == nil {
				localCabOrder.AckList = make(map[string]bool)
			}
			for ackID := range updatedCabOrder.AckList {
				localCabOrder.AckList[ackID] = true
			}
			localCabOrder.AckList[localWorldView.ID] = true
			if localCabOrder.StateofOrder == configuration.UnConfirmed { // Handle barrier condition: transition from UNCONFIRMED to CONFIRMED
				allAcknowledged := true // Check if all alive elevators have acknowledged this order
				for _, ackID := range IDsAliveElevators {
					if !localCabOrder.AckList[ackID] {
						allAcknowledged = false
						break
					}
				}
				if allAcknowledged { // If all alive elevators have acknowledged, transition to CONFIRMED
					localCabOrder.StateofOrder = configuration.Confirmed
				}
			}
		}
	}
	return localWorldView
}
