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
	//fmt.Println("we have entered updateworldviewwithbutton")
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
	//fmt.Println("LocalWorldView etter buttonPressed: ", localWorldView)
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
	//fmt.Println("localWV i covertfunc: ", worldView)
	//worldviewen som kommer inni ConverttoBOOL har 1 på order og ikke 2 (confirmed), derfor kan den ikke komme seg videre
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
	//fmt.Println("boolMatrix: ", boolMatrix)
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
				Behavior:    single_elevator.ToString(elevState.Elev.Behaviour),
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

func AssignOrder(worldView WorldView, IDsAliveElevators []string) map[string][][2]bool {
	//fmt.Println("vi har ankommet assignorder")
	input := HRAInputFormatting(worldView, IDsAliveElevators)
	//fmt.Println("Input til HRA: ",input )
	outputAssigner := AssignerExecutable.Assigner(input)
	return outputAssigner
}

func MergeOrdersHall(localOrder configuration.OrderMsg, updatedOrder configuration.OrderMsg, localWorldView WorldView, updatedWorldView WorldView, IDsAliveElevators []string) {
	if localOrder.AckList == nil {
		localOrder.AckList = make(map[string]bool)
	}

	switch updatedOrder.StateofOrder {
	case configuration.None: //hvis vi får inn en order som er none = de vet ingenting, bare chiller her
	case configuration.UnConfirmed: //stuck på unconfirmed / pass på
		//har kun en heis, som legges til i acklisten, men klarer fortsatt ikke gå fra unconfirmed til confirmed
		if localOrder.StateofOrder == configuration.None || localOrder.StateofOrder == configuration.Completed {
			localOrder.StateofOrder = configuration.UnConfirmed
			localOrder.AckList[updatedWorldView.ID] = true
		}
		if localOrder.StateofOrder == configuration.UnConfirmed { //handle barrier condition
			localOrder.AckList[localWorldView.ID] = true
			//MERGE ACKLISTER - legge til alle aktive heiser på nettet
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
	case configuration.Confirmed: //skal aldri få en confirmed, pga barrieren
		if localOrder.StateofOrder == configuration.None {
			localOrder.StateofOrder = configuration.Confirmed
		}
	case configuration.Completed:
		if localOrder.StateofOrder == configuration.None || localOrder.StateofOrder == configuration.Confirmed {
			localOrder.StateofOrder = configuration.Completed
		}
	}
}

func MergeOrdersCAB(localCABOrder configuration.OrderMsg, updatedCABOrder configuration.OrderMsg, localWorldView WorldView, updatedWorldView WorldView, IDsAliveElevators []string) {
	if localCABOrder.AckList == nil {
		localCABOrder.AckList = make(map[string]bool)
	}
	// for id := range updatedOrder.AckList {
	// 	localOrder.AckList[id] = true
	// }

	switch updatedCABOrder.StateofOrder {
	case configuration.None: //hvis vi får inn en order som er none = de vet ingenting, bare chiller her
	case configuration.UnConfirmed: //stuck på unconfirmed / pass på
		if localCABOrder.StateofOrder == configuration.None || localCABOrder.StateofOrder == configuration.Completed {
			localCABOrder.StateofOrder = configuration.UnConfirmed
			localCABOrder.AckList[updatedWorldView.ID] = true
		}
		if localCABOrder.StateofOrder == configuration.UnConfirmed { //handle barrier condition
			localCABOrder.AckList[localWorldView.ID] = true
			allAcknowledged := true
			for _, id := range IDsAliveElevators { // Check if all alive elevators have acknowledged this order
				if !localCABOrder.AckList[id] {
					allAcknowledged = false
					break
				}
			}
			if allAcknowledged { // If all alive elevators have acknowledged, transition to CONFIRMED
				localCABOrder.StateofOrder = configuration.Confirmed
			}
		}
	case configuration.Confirmed: //skal aldri få en confirmed, pga barrieren
		if localCABOrder.StateofOrder == configuration.None {
			localCABOrder.StateofOrder = configuration.Confirmed
		}
	case configuration.Completed:
		if localCABOrder.StateofOrder == configuration.None || localCABOrder.StateofOrder == configuration.Confirmed {
			localCABOrder.StateofOrder = configuration.Completed
		}
	}
}

// feil: localcaborder er kun en etasje. Får out of range error. Kan det være fordi localorder og updatedorder kun er 1 istedenfor 4 i størrelse?
func MergeWorldViews(localWorldView WorldView, updatedWorldView WorldView, IDsAliveElevators []string) WorldView {
	for id, state := range updatedWorldView.ElevatorStatusList { // Iterate over elevatorstatuslist in updatedWorldView and update the corresponding entries in the localWorldView
		localWorldView.ElevatorStatusList[id] = state
	}
	for floor := range localWorldView.HallOrderStatus { // Iterate over hall orders. Merge hallOrderStatus
		for button := range localWorldView.HallOrderStatus[floor] {
			// Get the local and updated orders for floor and button
			localOrder := &localWorldView.HallOrderStatus[floor][button]
			updatedOrder := updatedWorldView.HallOrderStatus[floor][button]
			MergeOrdersHall(*localOrder, updatedOrder, localWorldView, updatedWorldView, IDsAliveElevators)
		}
	}
	for id, elevState := range localWorldView.ElevatorStatusList { // Iterate over cab orders. Merge cab orders
		TotalLocalCabOrders = []bool{}
		TotalUpdatedCabOrders = []bool{}
		for floor := range elevState.Cab {
			localCabOrder := &localWorldView.ElevatorStatusList[id].Cab[floor]
			TotalLocalCabOrders = append(TotalLocalCabOrders, localCabOrder)
			updatedCabOrder := updatedWorldView.ElevatorStatusList[id].Cab[floor]
			TotalUpdatedCabOrders = append(TotalUpdatedCabOrders, updatedCabOrder)
			if localCabOrder.AckList == nil {
				localCabOrder.AckList = make(map[string]bool)
			}
			//MergeOrdersCAB(*localCabOrder, updatedCabOrder, localWorldView, updatedWorldView, IDsAliveElevators)
			MergeOrdersCAB(*TotalLocalCabOrders, TotalUpdatedCabOrders, localWorldView, updatedWorldView, IDsAliveElevators)

		}
	}
	return localWorldView
}
