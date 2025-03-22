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
			if localWorldView.HallOrderStatus[buttonPressed.Floor][buttonPressed.Button].StateofOrder == configuration.None {
				localWorldView.HallOrderStatus[buttonPressed.Floor][buttonPressed.Button] = configuration.OrderMsg{
					StateofOrder: configuration.UnConfirmed,
					AckList:      make(map[string]bool),
				}
				localWorldView.HallOrderStatus[buttonPressed.Floor][buttonPressed.Button].AckList[localWorldView.ID] = true
			} else {
				fmt.Println("Ignored button request, state: ", localWorldView.HallOrderStatus[buttonPressed.Floor][buttonPressed.Button])
			}
		case elevio.BT_Cab:
			if localWorldView.ElevatorStatusList[localWorldView.ID].Cab[buttonPressed.Floor].StateofOrder == configuration.None {
				localWorldView.ElevatorStatusList[localWorldView.ID].Cab[buttonPressed.Floor] = configuration.OrderMsg{
					StateofOrder: configuration.UnConfirmed,
					AckList:      make(map[string]bool),
				}
				localWorldView.ElevatorStatusList[localWorldView.ID].Cab[buttonPressed.Floor].AckList[localWorldView.ID] = true
			} else {
				fmt.Println("Ignored button request, state: ", localWorldView.ElevatorStatusList[localWorldView.ID].Cab[buttonPressed.Floor])
			}
		}
	} else {
		fmt.Println("Done with order: ", buttonPressed)
		switch buttonPressed.Button {
		case elevio.BT_HallUp, elevio.BT_HallDown:
			if localWorldView.HallOrderStatus[buttonPressed.Floor][buttonPressed.Button].StateofOrder == configuration.Confirmed {
				localWorldView.HallOrderStatus[buttonPressed.Floor][buttonPressed.Button].StateofOrder = configuration.Completed
				ResetAckList(localWorldView)
			} else {
				fmt.Println("Tried to clear button which was not confirmed: ", localWorldView.HallOrderStatus[buttonPressed.Floor][buttonPressed.Button])
			}
		case elevio.BT_Cab:
			if localWorldView.ElevatorStatusList[localWorldView.ID].Cab[buttonPressed.Floor].StateofOrder == configuration.Confirmed {
				localWorldView.ElevatorStatusList[localWorldView.ID].Cab[buttonPressed.Floor].StateofOrder = configuration.Completed
				ResetAckList(localWorldView)
			} else {
				fmt.Println("Tried to clear button not confirmed: ", localWorldView.ElevatorStatusList[localWorldView.ID].Cab[buttonPressed.Floor])
			}
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
	hallRequests := ConvertHallOrderStatestoBool(worldView)            // Konverter hallbestillinger til en boolsk matrise
	elevatorStates := make(map[string]AssignerExecutable.HRAElevState) // Opprett en map for å lagre tilstanden til hver heis
	for _, elevatorID := range IDsAliveElevators {                     // Gå gjennom alle aktive heiser
		elevStateMsg, exists := worldView.ElevatorStatusList[elevatorID] // Hent tilstanden til heisen fra ElevatorStatusList
		if !exists {
			continue // Hvis heisen ikke finnes i listen, hopp over
		}
		elevState := elevStateMsg.Elev // Hent tilstanden til heisen (Elevator-structen)
		if !elevState.Unavailable {    // Hvis heisen er tilgjengelig (ikke "Unavailable"), oppdater tilstanden
			cabRequests := make([]bool, configuration.NumFloors) // Opprett en liste for cab-bestillinger
			for floor, cabOrder := range elevStateMsg.Cab {
				cabRequests[floor] = cabOrder.StateofOrder == configuration.Confirmed // Sett cab-bestillinger til true hvis bestillingen er bekreftet
			}

			// Opprett en HRAElevState-struct for denne heisen
			elevatorStates[elevatorID] = AssignerExecutable.HRAElevState{
				Behavior:    single_elevator.ToString(elevState.Behaviour),                  // Konverter Behaviour til streng
				Floor:       elevState.Floor,                                                // Hent etasjen
				Direction:   elevio.DirToString(elevio.MotorDirection(elevState.Direction)), // Konverter Direction til streng
				CabRequests: cabRequests,                                                    // Legg til cab-bestillinger
			}
		}
	}

	// Returner HRAInput-structen med hall-bestillinger og heis-tilstander
	return AssignerExecutable.HRAInput{
		HallRequests: hallRequests,
		States:       elevatorStates,
	}
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
	// if
	// IDsAliveElevators = []string{worldView.ID}

	//fmt.Println("LOCAL WV ", worldView.HallOrderStatus)

	input := HRAInputFormatting(worldView, IDsAliveElevators)
	fmt.Println("Input til HRA: ", input)
	outputAssigner := AssignerExecutable.Assigner(input)
	return outputAssigner
}

//BØR PRØVE Å FÅ MERGE HALL OG MERGECAB TIL EN FUNKSJON

// endre sånn at funksjonen returnerer den oppdaterte localorder i stede for å endre direkte i WV
// det enste jeg har gjort er å oprette variablen updatedLocalhall som i starten er en kopi av localOrder og deretter er det denne variablen som endres
// så returneres updatedLocalhall
/*
func MergeOrdersHall(localOrder *configuration.OrderMsg, updatedOrder configuration.OrderMsg, localWorldView *WorldView, updatedWorldView WorldView, IDsAliveElevators []string) configuration.OrderMsg {
	updatedLocalHall := *localOrder //ENDRE NAVN, ER FORVIRRENDE

	if updatedLocalHall.AckList == nil {
		updatedLocalHall.AckList = make(map[string]bool)
	}

	switch updatedOrder.StateofOrder {
	case configuration.None: //hvis vi får inn en order som er none = de vet ingenting, bare chiller her
	case configuration.UnConfirmed: //stuck på unconfirmed / pass på
		if updatedLocalHall.StateofOrder == configuration.None { //|| updatedLocalHall.StateofOrder == configuration.Completed
			updatedLocalHall.StateofOrder = configuration.UnConfirmed
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
		//NONE barrier
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

	if order_new > current_order {
		current_order = order_new;
		current_order.AckList[ourId] = true
	}

	// should we convert to new order or keep our order
	// with our new local order, does it satisfy the requirements for the barriers

	//her håndterer vi states
	switch localCABOrder.StateofOrder {
	case configuration.None:
		if updatedCABOrder.StateofOrder != configuration.Completed {
			updatedLocalCab.StateofOrder = updatedCABOrder.StateofOrder
			updatedLocalCab.AckList = updatedCABOrder.AckList
			updatedLocalCab.AckList[localWorldView.ID] = true
		}
	case configuration.UnConfirmed:
		if updatedCABOrder.StateofOrder == configuration.Confirmed || updatedCABOrder.StateofOrder == configuration.Completed {
			// set our order to recieved order... add ourselves to acklist
		} else if updatedCABOrder.StateofOrder == configuration.UnConfirmed {
			// merge both acklists
		}
	}

	//her håndterier vi barrier
	if updatedLocalCab.StateofOrder == configuration.UnConfirmed {
		if fullacklist(...) {
			updatedLocalCab.StateofOrder = configuration.Confirmed;
			resetacklit();
		}
	} else if updatedLocalCab.StateofOrder == configuration.Completed {
		/// same
	}


	//gammel kode, skal ikke være her
	switch updatedCABOrder.StateofOrder {
	case configuration.None: //hvis vi får inn en order som er none = de vet ingenting, bare chiller her
	case configuration.UnConfirmed: //stuck på unconfirmed / pass på
		if updatedLocalCab.StateofOrder == configuration.None { //|| updatedLocalCab.StateofOrder == configuration.Completed
			updatedLocalCab.StateofOrder = configuration.UnConfirmed
			//updatedLocalCab.AckList[updatedWorldView.ID] = true
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
*/

func MergeWorldViews(localWorldView *WorldView, receivedWorldView WorldView, IDsAliveElevators []string) WorldView {
	MergedWorldView := *localWorldView
	//fmt.Println("LOCAL WORLD VIEW", localWorldView.ElevatorStatusList)
	//fmt.Println("RECEIVED WORLD VIEW", receivedWorldView.ElevatorStatusList)
	//fmt.Println("Hall: ", localWorldView.HallOrderStatus)

	// Oppdater tilstanden til hver heis i receivedWorldView
	for id, elevState := range receivedWorldView.ElevatorStatusList {
		// Hvis heisen finnes i receivedWorldView, oppdater tilstanden
		if _, exists := localWorldView.ElevatorStatusList[id]; exists {
			localWorldView.ElevatorStatusList[id] = elevState
		}
	}

	for floor := range localWorldView.HallOrderStatus { // Iterate over hall orders. Merge hallOrderStatus
		for button := range localWorldView.HallOrderStatus[floor] {
			// Get the local and updated orders for floor and button
			localOrder := &localWorldView.HallOrderStatus[floor][button]
			receivedOrder := receivedWorldView.HallOrderStatus[floor][button]
			HallOrderMerged := MergeOrders(localOrder, receivedOrder, localWorldView, receivedWorldView, IDsAliveElevators)
			MergedWorldView.HallOrderStatus[floor][button] = HallOrderMerged
			if HallOrderMerged.StateofOrder == configuration.Completed {
				fmt.Println("Complete, floor: ", floor, " button: ", button, "recieved: ", receivedOrder)
			}
		}
	}

	for id, elevState := range receivedWorldView.ElevatorStatusList {

		_, localElevStateExists := localWorldView.ElevatorStatusList[id] //Sjekker om id finnes som en nøkkel i localWorldView.ElevatorStatusList

		if !localElevStateExists { //Sjekker om id ikke finnes i localWorldView.ElevatorStatusList, og hvis det ikke finnes, legger det til elevState i mappen.
			localWorldView.ElevatorStatusList[id] = elevState
		} else {
			//fmt.Println("Id: ", id, " Cab: ", localWorldView.ElevatorStatusList[id].Cab)
			//fmt.Println("IdsAlive: ", IDsAliveElevators)

			for floor := range elevState.Cab {
				localCabOrder := &localWorldView.ElevatorStatusList[id].Cab[floor]
				receivedOrder := receivedWorldView.ElevatorStatusList[id].Cab[floor]

				if localCabOrder.AckList == nil {
					localCabOrder.AckList = make(map[string]bool)
				}

				//MergeOrdersCAB(*localCabOrder, updatedCabOrder, localWorldView, updatedWorldView, IDsAliveElevators)
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
	//cyclic counter - should we convert to new order or keep our order
	/*
		if order_new > current_order {
		current_order = order_new;
		current_order.AckList[ourId] = true
		}
	*/
	// gotUnconFromA := false
	switch updatedLocalOrder.StateofOrder {
	case configuration.None:
		if receivedOrder.StateofOrder != configuration.Completed {
			updatedLocalOrder.StateofOrder = receivedOrder.StateofOrder
			updatedLocalOrder.AckList = receivedOrder.AckList
			updatedLocalOrder.AckList[localWorldView.ID] = true
		}
	case configuration.UnConfirmed:
		fmt.Println("Is unconfirmed: ", receivedOrder)
		if receivedOrder.StateofOrder == configuration.Confirmed || receivedOrder.StateofOrder == configuration.Completed {
			//set our order to received order
			//add ourselves to acklist
			updatedLocalOrder.StateofOrder = receivedOrder.StateofOrder
			updatedLocalOrder.AckList = receivedOrder.AckList
			updatedLocalOrder.AckList[localWorldView.ID] = true
		} else if receivedOrder.StateofOrder == configuration.UnConfirmed {
			// if receivedOrder.AckList["A"] && localWorldView.ID != "A" {
			// 	fmt.Println("Got unconfirmed from A")
			// 	gotUnconFromA = true
			// }

			// Merge AckLists
			for id, acknowledged := range receivedOrder.AckList {
				// Hvis noden har bekreftet bestillingen, legg den til i den oppdaterte listen
				if acknowledged {
					updatedLocalOrder.AckList[id] = true
				}
			}
			// Sørg for at vi selv er i AckList
			updatedLocalOrder.AckList[localWorldView.ID] = true
		}
	case configuration.Confirmed: //stemmer det at vi ikke trenger å oppdatere acklisten her?
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
			// Merge AckLists
			for id, acknowledged := range receivedOrder.AckList {
				// Hvis noden har bekreftet bestillingen, legg den til i den oppdaterte listen
				if acknowledged {
					updatedLocalOrder.AckList[id] = true
				}
			}
			// Sørg for at vi selv er i AckList
			updatedLocalOrder.AckList[localWorldView.ID] = true
		}
	}
	//håndtere barrier etter switch - with our new local order, does it satisfy the requirements for the barriers
	if updatedLocalOrder.StateofOrder == configuration.UnConfirmed {
		allAcknowledged := true
		for _, id := range IDsAliveElevators { // Check if all alive elevators have acknowledged this order
			if !updatedLocalOrder.AckList[id] {
				//fmt.Println("Id not in acklist: ", id)
				allAcknowledged = false
				break
			}
		}
		//fmt.Println("All acks: ", allAcknowledged)
		if allAcknowledged { // If all alive elevators have acknowledged, transition to CONFIRMED
			fmt.Println("I confirmed order")
			updatedLocalOrder.StateofOrder = configuration.Confirmed
			//fmt.Println("Order CONFIRMED")
			ResetAckList(localWorldView)
		}
	} else if updatedLocalOrder.StateofOrder == configuration.Completed {
		allAcknowledged := true
		for _, id := range IDsAliveElevators { // Check if all alive elevators have acknowledged this order
			if !updatedLocalOrder.AckList[id] {
				//fmt.Println("Id not in acklist: ", id)
				allAcknowledged = false
				fmt.Println("Did not find: ", id)
				break
			}
		}
		//fmt.Println("All acks: ", allAcknowledged)
		if allAcknowledged { // If all alive elevators have acknowledged, transition to CONFIRMED
			updatedLocalOrder.StateofOrder = configuration.None
			//fmt.Println("Order set to NONE")
			ResetAckList(localWorldView)
		}
	}

	// if gotUnconFromA {
	// 	if updatedLocalOrder.StateofOrder == configuration.Confirmed {
	// 		fmt.Println("Good Confirmed")
	// 	} else {
	// 		panic("Did not confirm order??")
	// 	}
	// }
	return updatedLocalOrder
}
