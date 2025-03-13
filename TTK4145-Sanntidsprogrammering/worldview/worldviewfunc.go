package worldview

import (
	"TTK4145-Heislab/AssignerExecutable"
	"TTK4145-Heislab/configuration"
	"TTK4145-Heislab/driver-go/elevio"
	"TTK4145-Heislab/single_elevator"
	"fmt"
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

/*
func MergeOrdersHall(localOrder *configuration.OrderMsg, updatedOrder configuration.OrderMsg, localWorldView WorldView, updatedWorldView WorldView, IDsAliveElevators []string) {
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

func MergeOrdersCAB(localCABOrder *configuration.OrderMsg, updatedCABOrder configuration.OrderMsg, localWorldView WorldView, updatedWorldView WorldView, IDsAliveElevators []string) configuration.OrderMsg {

	if localCABOrder.AckList == nil {
		localCABOrder.AckList = make(map[string]bool)
	}

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
					fmt.Println("Id not in acklist: ", id)
					allAcknowledged = false
					break
				}
			}
			fmt.Println("All acks: ", allAcknowledged)
			if allAcknowledged { // If all alive elevators have acknowledged, transition to CONFIRMED
				localCABOrder.StateofOrder = configuration.Confirmed
				fmt.Println("Order confirmed")
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
*/

//BØR PRØVE Å FÅ MERGE HALL OG MERGECAB TIL EN FUNKSJON

// endre sånn at funksjonen returnerer den oppdaterte localorder i stede for å endre direkte i WV
// det enste jeg har gjort er å oprette variablen updatedLocalhall som i starten er en kopi av localOrder og deretter er det denne variablen som endres
// så returneres updatedLocalhall
func MergeOrdersHall(localOrder *configuration.OrderMsg, updatedOrder configuration.OrderMsg, localWorldView *WorldView, updatedWorldView WorldView, IDsAliveElevators []string) configuration.OrderMsg {
	updatedLocalHall := *localOrder //ENDRE NAVN, ER FORVIRRENDE

	if updatedLocalHall.AckList == nil {
		updatedLocalHall.AckList = make(map[string]bool)
	}

	switch updatedOrder.StateofOrder {
	case configuration.None: //hvis vi får inn en order som er none = de vet ingenting, bare chiller her
	case configuration.UnConfirmed: //stuck på unconfirmed / pass på
		//har kun en heis, som legges til i acklisten, men klarer fortsatt ikke gå fra unconfirmed til confirmed
		if updatedLocalHall.StateofOrder == configuration.None || updatedLocalHall.StateofOrder == configuration.Completed {
			updatedLocalHall.StateofOrder = configuration.UnConfirmed
			updatedLocalHall.AckList[updatedWorldView.ID] = true
		}
		if updatedLocalHall.StateofOrder == configuration.UnConfirmed { //handle barrier condition
			updatedLocalHall.AckList[localWorldView.ID] = true
			//MERGE ACKLISTER - legge til alle aktive heiser på nettet
			allAcknowledged := true
			for _, id := range IDsAliveElevators { // Check if all alive elevators have acknowledged this order
				if !updatedLocalHall.AckList[id] {
					allAcknowledged = false
					break
				}
			}
			if allAcknowledged { // If all alive elevators have acknowledged, transition to CONFIRMED
				updatedLocalHall.StateofOrder = configuration.Confirmed
			}
		}
	case configuration.Confirmed: //skal aldri få en confirmed, pga barrieren
		// fmt.Println("Helloooooooo, my order state: ", updatedLocalHall.StateofOrder)
		if updatedLocalHall.StateofOrder == configuration.None || updatedLocalHall.StateofOrder == configuration.UnConfirmed {
			updatedLocalHall.StateofOrder = configuration.Confirmed
		}
	case configuration.Completed:
		if updatedLocalHall.StateofOrder == configuration.None || updatedLocalHall.StateofOrder == configuration.Confirmed {
			updatedLocalHall.StateofOrder = configuration.Completed
		}
	}
	return updatedLocalHall
}

// endre sånn at funksjonen returnerer den oppdaterte localCABorder i stede for å endre direkte i WV
// det enste jeg har gjort er å oprette variablen updatedLocalCab som i starten er en kopi av localCABOrder og deretter er det denne variablen som endres
// så returneres updatedLocalCab
func MergeOrdersCAB(localCABOrder *configuration.OrderMsg, updatedCABOrder configuration.OrderMsg, localWorldView *WorldView, updatedWorldView WorldView, IDsAliveElevators []string) configuration.OrderMsg {
	updatedLocalCab := *localCABOrder

	if updatedLocalCab.AckList == nil {
		updatedLocalCab.AckList = make(map[string]bool)
	}
	// for id := range updatedOrder.AckList {
	// 	localOrder.AckList[id] = true
	// }

	switch updatedCABOrder.StateofOrder {
	case configuration.None: //hvis vi får inn en order som er none = de vet ingenting, bare chiller her
	case configuration.UnConfirmed: //stuck på unconfirmed / pass på
		if updatedLocalCab.StateofOrder == configuration.None || updatedLocalCab.StateofOrder == configuration.Completed {
			updatedLocalCab.StateofOrder = configuration.UnConfirmed
			updatedLocalCab.AckList[updatedWorldView.ID] = true
		}
		if updatedLocalCab.StateofOrder == configuration.UnConfirmed { //handle barrier condition
			updatedLocalCab.AckList[localWorldView.ID] = true
			allAcknowledged := true
			for _, id := range IDsAliveElevators { // Check if all alive elevators have acknowledged this order
				if !updatedLocalCab.AckList[id] {
					fmt.Println("Id not in acklist: ", id)
					allAcknowledged = false
					break
				}
			}
			fmt.Println("All acks: ", allAcknowledged)
			if allAcknowledged { // If all alive elevators have acknowledged, transition to CONFIRMED
				updatedLocalCab.StateofOrder = configuration.Confirmed
				fmt.Println("Order confirmed")
			}
		}
	case configuration.Confirmed: //skal aldri få en confirmed, pga barrieren
		if updatedLocalCab.StateofOrder == configuration.None || updatedLocalCab.StateofOrder == configuration.UnConfirmed {
			updatedLocalCab.StateofOrder = configuration.Confirmed
		}
	case configuration.Completed:
		if updatedLocalCab.StateofOrder == configuration.None || localCABOrder.StateofOrder == configuration.Confirmed {
			updatedLocalCab.StateofOrder = configuration.Completed
		}
	}
	return updatedLocalCab
}

func MergeWorldViews(localWorldView *WorldView, updatedWorldView WorldView, IDsAliveElevators []string) WorldView {
	MergedWorldView := *localWorldView

	fmt.Println("Hall: ", localWorldView.HallOrderStatus)
	for floor := range localWorldView.HallOrderStatus { // Iterate over hall orders. Merge hallOrderStatus
		for button := range localWorldView.HallOrderStatus[floor] {
			// Get the local and updated orders for floor and button
			localOrder := &localWorldView.HallOrderStatus[floor][button]
			updatedOrder := updatedWorldView.HallOrderStatus[floor][button]
			HallOrderMerged := MergeOrdersHall(localOrder, updatedOrder, localWorldView, updatedWorldView, IDsAliveElevators)
			MergedWorldView.HallOrderStatus[floor][button] = HallOrderMerged
		}
	}

	for id, elevState := range updatedWorldView.ElevatorStatusList {

		_, localElevStateExists := localWorldView.ElevatorStatusList[id] //Sjekker om id finnes som en nøkkel i localWorldView.ElevatorStatusList

		if !localElevStateExists { //Sjekker om id ikke finnes i localWorldView.ElevatorStatusList, og hvis det ikke finnes, legger det til elevState i mappen.
			localWorldView.ElevatorStatusList[id] = elevState
		} else {
			fmt.Println("Id: ", id, " Cab: ", localWorldView.ElevatorStatusList[id].Cab)
			fmt.Println("IdsAlive: ", IDsAliveElevators)

			for floor := range elevState.Cab {
				localCabOrder := &localWorldView.ElevatorStatusList[id].Cab[floor]
				updatedCabOrder := updatedWorldView.ElevatorStatusList[id].Cab[floor]

				if localCabOrder.AckList == nil {
					localCabOrder.AckList = make(map[string]bool)
				}

				//MergeOrdersCAB(*localCabOrder, updatedCabOrder, localWorldView, updatedWorldView, IDsAliveElevators)
				CabOrderMerged := MergeOrdersCAB(localCabOrder, updatedCabOrder, localWorldView, updatedWorldView, IDsAliveElevators)
				MergedWorldView.ElevatorStatusList[id].Cab[floor] = CabOrderMerged
			}
		}
	}
	return MergedWorldView
}
