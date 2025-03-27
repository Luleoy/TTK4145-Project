package worldView

import (
	"TTK4145-Heislab/assignerExecutable"
	"TTK4145-Heislab/configuration"
	"TTK4145-Heislab/driver-go/elevio"
	"TTK4145-Heislab/singleElevator"
	"fmt"
	"time"
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

func InitializeCabOrderStatus() []configuration.OrderMsg {
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
		Elev: singleElevator.Elevator{},
		Cab:  InitializeCabOrderStatus(),
	}
	wv.ElevatorStatusList[elevatorID] = elevatorState
	return wv
}

func DetermineInitialDirection(WorldViewRXChannel <-chan WorldView, elevatorID string) elevio.MotorDirection {
	select {
	case worldView := <-WorldViewRXChannel:
		if status, exists := worldView.ElevatorStatusList[elevatorID]; exists {
			if status.Elev.Direction == singleElevator.Down {
				return elevio.MD_Down
			}
			return elevio.MD_Up
		}
	case <-time.After(100 * time.Millisecond):
	}
	return elevio.MD_Down // Standard retning ved timeout
}

func UpdateWorldViewWithButton(localWorldView *WorldView, button elevio.ButtonEvent, isNewOrder bool) WorldView {
	updatedLocalWorldView := *localWorldView
	if isNewOrder { //Updating order from StateofOrder None to Unconfirmed (received new button press)
		switch button.Button {
		case elevio.BT_HallUp, elevio.BT_HallDown:
			if updatedLocalWorldView.HallOrderStatus[button.Floor][button.Button].StateofOrder == configuration.None {
				updatedLocalWorldView.HallOrderStatus[button.Floor][button.Button] = configuration.OrderMsg{
					StateofOrder: configuration.UnConfirmed,
					AckList:      make(map[string]bool),
				}
				updatedLocalWorldView.HallOrderStatus[button.Floor][button.Button].AckList[updatedLocalWorldView.ID] = true
			} else {
				fmt.Println("Ignored button request, state: ", updatedLocalWorldView.HallOrderStatus[button.Floor][button.Button])
			}
		case elevio.BT_Cab:
			if updatedLocalWorldView.ElevatorStatusList[updatedLocalWorldView.ID].Cab[button.Floor].StateofOrder == configuration.None {
				updatedLocalWorldView.ElevatorStatusList[updatedLocalWorldView.ID].Cab[button.Floor] = configuration.OrderMsg{
					StateofOrder: configuration.UnConfirmed,
					AckList:      make(map[string]bool),
				}
				updatedLocalWorldView.ElevatorStatusList[updatedLocalWorldView.ID].Cab[button.Floor].AckList[updatedLocalWorldView.ID] = true
			} else {
				fmt.Println("Ignored button request, state: ", updatedLocalWorldView.ElevatorStatusList[updatedLocalWorldView.ID].Cab[button.Floor])
			}
		}
	} else { //Updating order from StateofOrder Confirmed to Completed (executed by single elevator)
		switch button.Button {
		case elevio.BT_HallUp, elevio.BT_HallDown:
			if updatedLocalWorldView.HallOrderStatus[button.Floor][button.Button].StateofOrder == configuration.Confirmed {
				updatedLocalWorldView.HallOrderStatus[button.Floor][button.Button].StateofOrder = configuration.Completed
				ResetAckList(&updatedLocalWorldView)
			} else {
				fmt.Println("Tried to clear button which was not confirmed: ", updatedLocalWorldView.HallOrderStatus[button.Floor][button.Button])
			}
		case elevio.BT_Cab:
			if updatedLocalWorldView.ElevatorStatusList[updatedLocalWorldView.ID].Cab[button.Floor].StateofOrder == configuration.Confirmed {
				updatedLocalWorldView.ElevatorStatusList[updatedLocalWorldView.ID].Cab[button.Floor].StateofOrder = configuration.Completed
				ResetAckList(&updatedLocalWorldView)
			} else {
				fmt.Println("Tried to clear button not confirmed: ", updatedLocalWorldView.ElevatorStatusList[updatedLocalWorldView.ID].Cab[button.Floor])
			}
		}
	}
	return updatedLocalWorldView
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

func HRAInputFormatting(worldView WorldView, IDsAliveElevators []string) assignerExecutable.HRAInput {
	hallRequests := ConvertHallOrderStatestoBool(worldView)
	elevatorStates := make(map[string]assignerExecutable.HRAElevState)

	for _, elevatorID := range IDsAliveElevators {
		elevStateMsg, exists := worldView.ElevatorStatusList[elevatorID]
		if !exists {
			continue
		}
		cabRequests := make([]bool, configuration.NumFloors)
		for floor, cabOrder := range elevStateMsg.Cab {
			cabRequests[floor] = cabOrder.StateofOrder == configuration.Confirmed
		}
		elevatorStates[elevatorID] = assignerExecutable.HRAElevState{
			Behavior:    singleElevator.BehaviourToString(elevStateMsg.Elev.Behaviour),
			Floor:       elevStateMsg.Elev.Floor,
			Direction:   elevio.DirToString(elevio.MotorDirection(elevStateMsg.Elev.Direction)),
			CabRequests: cabRequests,
		}
	}
	return assignerExecutable.HRAInput{
		HallRequests: hallRequests,
		States:       elevatorStates,
	}
}

func MergeCABandHRAOutput(OurHall [][2]bool, Ourcab []bool) singleElevator.Orders {
	var OrderMatrix singleElevator.Orders
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

func GetCAB(localWorldView WorldView, ID string) []bool {
	cabOrders := localWorldView.ElevatorStatusList[ID].Cab
	Cab := make([]bool, len(cabOrders))
	for i, order := range cabOrders {
		Cab[i] = order.StateofOrder == configuration.Confirmed
	}
	return Cab
}

func SetLights(localWorldView WorldView) {
	for floor := range localWorldView.HallOrderStatus {
		for button := 0; button < 2; button++ {
			order := localWorldView.HallOrderStatus[floor][button]
			Light := order.StateofOrder == configuration.Confirmed
			elevio.SetButtonLamp(elevio.ButtonType(button), floor, Light)
		}
	}
	for id, elevatorState := range localWorldView.ElevatorStatusList {
		if id == localWorldView.ID {
			for floor, order := range elevatorState.Cab {
				Light := order.StateofOrder == configuration.Confirmed
				elevio.SetButtonLamp(elevio.BT_Cab, floor, Light)
			}
		}
	}
}

func AssignOrder(worldView WorldView, IDsAliveElevators []string) map[string][][2]bool {
	input := HRAInputFormatting(worldView, IDsAliveElevators)
	outputAssigner := assignerExecutable.Assigner(input)
	return outputAssigner
}

func MergeWorldViews(localWorldView *WorldView, receivedWorldView WorldView, IDsAliveElevators []string) WorldView {
	MergedWorldView := *localWorldView

	//TOR -review det Nils har sendt
	if _, exists := localWorldView.ElevatorStatusList[receivedWorldView.ID]; exists {
		currentElevState := localWorldView.ElevatorStatusList[receivedWorldView.ID]
		currentElevState.Elev = receivedWorldView.ElevatorStatusList[receivedWorldView.ID].Elev
		localWorldView.ElevatorStatusList[receivedWorldView.ID] = currentElevState
	} else {
		localWorldView.ElevatorStatusList[receivedWorldView.ID] = receivedWorldView.ElevatorStatusList[receivedWorldView.ID]
	}

	for floor := range localWorldView.HallOrderStatus {
		for button := range localWorldView.HallOrderStatus[floor] {
			localOrder := &localWorldView.HallOrderStatus[floor][button]
			receivedOrder := receivedWorldView.HallOrderStatus[floor][button]
			HallOrderMerged := MergeOrders(localOrder, receivedOrder, localWorldView, receivedWorldView, IDsAliveElevators)
			MergedWorldView.HallOrderStatus[floor][button] = HallOrderMerged
		}
	}

	for id, elevState := range receivedWorldView.ElevatorStatusList {
		_, localElevStateExists := localWorldView.ElevatorStatusList[id]

		if !localElevStateExists {
			localWorldView.ElevatorStatusList[id] = elevState
		} else {

			for floor := range elevState.Cab {
				localCabOrder := &localWorldView.ElevatorStatusList[id].Cab[floor]
				receivedOrder := receivedWorldView.ElevatorStatusList[id].Cab[floor]

				if localCabOrder.AckList == nil {
					localCabOrder.AckList = make(map[string]bool)
				}

				CabOrderMerged := MergeOrders(localCabOrder, receivedOrder, localWorldView, receivedWorldView, IDsAliveElevators)
				MergedWorldView.ElevatorStatusList[id].Cab[floor] = CabOrderMerged
			}
		}
	}
	return MergedWorldView
}

func MergeOrders(localOrder *configuration.OrderMsg, receivedOrder configuration.OrderMsg, localWorldView *WorldView, updatedWorldView WorldView, IDsAliveElevators []string) configuration.OrderMsg {
	updatedLocalOrder := *localOrder
	if updatedLocalOrder.AckList == nil {
		updatedLocalOrder.AckList = make(map[string]bool)
	}
	switch updatedLocalOrder.StateofOrder { //Switch case handles cyclic counter with StateofOrder; None, Unconfirmed, Confirmed and Completed.
	case configuration.None:
		if receivedOrder.StateofOrder != configuration.Completed {
			updatedLocalOrder.StateofOrder = receivedOrder.StateofOrder
			updatedLocalOrder.AckList = receivedOrder.AckList
			updatedLocalOrder.AckList[localWorldView.ID] = true
		}
	case configuration.UnConfirmed:
		if receivedOrder.StateofOrder == configuration.Confirmed || receivedOrder.StateofOrder == configuration.Completed {
			updatedLocalOrder.StateofOrder = receivedOrder.StateofOrder
			updatedLocalOrder.AckList = receivedOrder.AckList
			updatedLocalOrder.AckList[localWorldView.ID] = true
		} else if receivedOrder.StateofOrder == configuration.UnConfirmed {
			for id, acknowledged := range receivedOrder.AckList {
				if acknowledged {
					updatedLocalOrder.AckList[id] = true
				}
			}
			updatedLocalOrder.AckList[localWorldView.ID] = true
		}
	case configuration.Confirmed:
		if receivedOrder.StateofOrder == configuration.Completed {
			updatedLocalOrder.StateofOrder = receivedOrder.StateofOrder
			updatedLocalOrder.AckList = receivedOrder.AckList
			updatedLocalOrder.AckList[localWorldView.ID] = true
		}
	case configuration.Completed:
		if receivedOrder.StateofOrder == configuration.None {
			updatedLocalOrder.StateofOrder = configuration.None
			updatedLocalOrder.AckList = receivedOrder.AckList
			updatedLocalOrder.AckList[localWorldView.ID] = true
		} else if receivedOrder.StateofOrder == configuration.Completed {
			for id, acknowledged := range receivedOrder.AckList {
				if acknowledged {
					updatedLocalOrder.AckList[id] = true
				}
			}
			updatedLocalOrder.AckList[localWorldView.ID] = true
		}
	}
	if updatedLocalOrder.StateofOrder == configuration.UnConfirmed { //Handling of barrier condition from StateofOrder Unconfirmed to Confirmed
		allAcknowledged := true
		for _, id := range IDsAliveElevators {
			if !updatedLocalOrder.AckList[id] {
				allAcknowledged = false
				break
			}
		}
		if allAcknowledged {
			updatedLocalOrder.StateofOrder = configuration.Confirmed
			ResetAckList(localWorldView)
		}
	} else if updatedLocalOrder.StateofOrder == configuration.Completed { ////Handling of barrier condition from StateofOrder Completed to None
		allAcknowledged := true
		for _, id := range IDsAliveElevators {
			if !updatedLocalOrder.AckList[id] {
				allAcknowledged = false
				break
			}
		}
		if allAcknowledged {
			updatedLocalOrder.StateofOrder = configuration.None
			ResetAckList(localWorldView)
		}
	}
	return updatedLocalOrder
}

func UpdateLastChanged(localWorldView WorldView, receivedWorldView WorldView, currentLastChanged map[string]time.Time) map[string]time.Time {
	newLastChanged := make(map[string]time.Time)
	for id, val := range currentLastChanged {
		newLastChanged[id] = val
	}

	if _, exists := newLastChanged[receivedWorldView.ID]; !exists {
		newLastChanged[receivedWorldView.ID] = time.Now()
	} else if localWorldView.ElevatorStatusList[receivedWorldView.ID].Elev.Behaviour != receivedWorldView.ElevatorStatusList[receivedWorldView.ID].Elev.Behaviour ||
		localWorldView.ElevatorStatusList[receivedWorldView.ID].Elev.Floor != receivedWorldView.ElevatorStatusList[receivedWorldView.ID].Elev.Floor ||
		receivedWorldView.ElevatorStatusList[receivedWorldView.ID].Elev.Behaviour == singleElevator.Idle {
		newLastChanged[receivedWorldView.ID] = time.Now()
	}
	return newLastChanged
}
