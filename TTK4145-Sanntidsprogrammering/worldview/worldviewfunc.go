package worldview

import (
	"TTK4145-Heislab/AssignerExecutable" //har dette noe med go.mod filen i assignerexecutable??
	"TTK4145-Heislab/configuration"
	"TTK4145-Heislab/driver-go/elevio"
	"TTK4145-Heislab/single_elevator"
)

//HVA MÅ GJØRES???
//loope gjennom Cab buttons også i mergeworldviews
//resette acklist ved OrderState endring
//mismatch type - hallorderstatus og elevatorstatuslist

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

// ELEVATOR ID - legge til cab på bestemt ID
func updateWorldViewWithButton(localWorldView *WorldView, buttonPressed elevio.ButtonEvent, isNewOrder bool) WorldView {
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
		case elevio.BT_Cab:
			localWorldView.ElevatorStatusList[localWorldView.ID].Cab[buttonPressed.Floor].StateofOrder = configuration.Completed
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

// oversette WorldView til HRAInput (tar inn WorldView og konverterer til format som kan brukes av HRA)
// merge hall request and cab requests
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

// ID på CAB buttons
// output fra assigner må appendes med CAB - MERGE
// MERGED må sendes til SINGLE ELEVATOR FSM
// må iterere gjennom keys og velge riktig elevator
// velge riktig elevator og sette en på riktig sted til dens ordermatrix
// MergeCABandHRA → Merges the converted HallOrderStatus with CabRequests to form a 4x3 matrix.
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

func AssignOrder(worldView WorldView, IDsAliveElevators []string) map[string][][2]bool {
	input := HRAInputFormatting(worldView, IDsAliveElevators)
	outputAssigner := AssignerExecutable.Assigner(input)
	return outputAssigner
}

func GetOurCAB(localWorldView WorldView, ourID string) []configuration.OrderMsg { //må man ha med ID her?
	return localWorldView.ElevatorStatusList[ourID].Cab
}

//output fra assigner - map av id og hvilke ordre som skal tas
//legge til cab orders som en kolonne på høyre side
//alt må være bools og en 4x3 matrise
//return ordermatrix

func MergeWorldViews(localWorldView WorldView, updatedWorldView WorldView, IDsAliveElevators []string) WorldView { //ikke iterert over CAB!!!!
	//sjekke hvor mange som er i live??? hva skal vi gjøre med den infoen
	//disse IDene må acknowledge og være i Acklist
	//alle må ha oppdatert worldview før den kan assignes og utføres

	//iterate over elevatorstatuslist in updatedworldview ad update the corresponding entries in the localworldview
	//den lokale verden får den nyeste informasjonen om alle heiser
	for id, state := range updatedWorldView.ElevatorStatusList {
		localWorldView.ElevatorStatusList[id] = state
	}

	//iterate over hallorders. merge hallorderstatus and handle the barrier condition
	for floor := range localWorldView.HallOrderStatus {
		for button := range localWorldView.HallOrderStatus[floor] {
			//get the local and updated orders for floor and button
			localOrder := &localWorldView.HallOrderStatus[floor][button]
			updatedOrder := updatedWorldView.HallOrderStatus[floor][button]

			if localOrder.AckList == nil {
				localOrder.AckList = make(map[string]bool)
			}

			//merge acklist for this order
			for id := range updatedOrder.AckList {
				localOrder.AckList[id] = true
			}

			//add elevator ID to acklist
			localOrder.AckList[localWorldView.ID] = true

			//handle barrier condition: transition from UNCONFIRMED to CONFIRMED
			if localOrder.StateofOrder == configuration.UnConfirmed {
				//check if all alive elevators have acknowledged this order
				allAcknowledged := true
				for _, id := range IDsAliveElevators {
					if !localOrder.AckList[id] {
						allAcknowledged = false
						break
					}
				}
				//if all alive elevators have acknowledged, transitionto CONFIRMED
				if allAcknowledged {
					localOrder.StateofOrder = configuration.Confirmed
				}
			}
		}
	}
	return localWorldView
}
