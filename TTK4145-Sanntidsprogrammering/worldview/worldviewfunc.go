package worldview

import (
	"TTK4145-Heislab/AssignerExecutable"
	"TTK4145-Heislab/configuration"
	"TTK4145-Heislab/driver-go/elevio"
	"TTK4145-Heislab/single_elevator"
)

// REQUESTSTATE INT - lage en funksjon som konverterer fra WorldView til OrderMatrix som vi kan bruke

func InitializeWorldView(elevatorID string) WorldView {
	message := WorldView{
		ID:                 elevatorID,
		Acklist:            make([]string, 0),
		ElevatorStatusList: map[string]single_elevator.State{elevatorID: single_elevator.State},
		HallOrderStatus:    InitializeHallOrderStatus(),
		//CabRequests?
	}
	return message
}

// funksjon som skal initialisere hallorderstatus. skal etterhvert ha false på alle utenom confirmed med true
func InitializeHallOrderStatus() [][configuration.NumButtons - 1]configuration.RequestState {
	HallOrderStatus := make([][configuration.NumButtons - 1]configuration.RequestState, configuration.NumFloors)
	for floor := range HallOrderStatus {
		for button := range HallOrderStatus[floor] {
			HallOrderStatus[floor][button] = configuration.None
		}
	}
	return HallOrderStatus
}

	//ELEVATOR ID - legge til cab på bestemt ID 
func updateWorldViewWithButton(localWorldView *WorldView, buttonPressed elevio.ButtonEvent, B bool) WorldView {
	if B == true { //mottar knappetrykk som ny bestilling (buttonpressedchannel)
		if buttonPressed == elevio.BT_HallDown || elevio.BT_HallUp {
			localWorldView.HallOrderStatus[buttonPressed.Floor][buttonPressed.Button] = configuration.Unconfirmed 
		} 
		if buttonPressed == elevio.BT_Cab { //her må worldview være local
			localWorldView.CabRequests[buttonPressed.Floor] = true 
		}
	} else { //sender tilbake knappetrykk fra FSM (ordercompletedchannel)
		if buttonPressed == elevio.BT_HallDown || elevio.BT_HallUp {
			localWorldView.HallOrderStatus[buttonPressed.Floor][buttonPressed.Button] = configuration.Completed
		} 
		if buttonPressed == elevio.BT_Cab { //her må worldview være local
			localWorldView.CabRequests[buttonPressed.Floor] = false 
		}
	}
	return localWorldView
}


func ResetAckList(localWorldView *WorldView) {
	localWorldView.Acklist = make([]string, 0)
	localWorldView.Acklist = append(localWorldView.Acklist, localWorldView.ID)
}

func ConvertHallOrderStatustoBool(WorldView WorldView) [][2]bool {
	// Opprett en fast strukturert slice med [2]bool per etasje
	boolMatrix := make([][2]bool, configuration.NumFloors)
	for floor := 0; floor < configuration.NumFloors; floor++ {
		for button := 0; button < 2; button++ { // Kun 2 knapper per etasje (opp/ned)
			if WorldView.HallOrderStatus[floor][button] == configuration.Confirmed {
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
func HRAInputFormatting(WorldView WorldView) AssignerExecutable.HRAInput {
	elevatorStates := make(map[string]AssignerExecutable.HRAElevState)
	hallrequests := ConvertHallOrderStatustoBool(WorldView)

	for ID := range WorldView.Acklist {
		if !WorldView.ElevatorStatusList[WorldView.Acklist[ID]].Unavailable { //har ikke en unavailable
			elevatorStates[WorldView.Acklist[ID]] = AssignerExecutable.HRAElevState{
				Behaviour: single_elevator.ToString(WorldView.ElevatorStatusList[WorldView.Acklist[ID]].Behaviour),
				Floor:     WorldView.ElevatorStatusList[WorldView.Acklist[ID]].Floor,
				Direction: elevio.DirToString(elevio.MotorDirection(WorldView.ElevatorStatusList[WorldView.Acklist[ID]].Direction)), //Direction: elevio.DirToString(WorldView.ElevatorStatusList[WorldView.Acklist[ID]].Direction),
				//CABREQUESTS - hvordan håndtere (HARAINput har CAB requests)
			}
		}
	}
	input := AssignerExecutable.HRAInput{
		HallRequests: hallrequests,
		States:       elevatorStates,
	}
	return input
}

func AssignOrder(WorldView WorldView, newOrderChannel chan<- single_elevator.Orders) { //map med ID som nøkkel, og arrays med 2 bolske verdier med orders true or false
	input := HRAInputFormatting(WorldView) //Konverterer WorldView til riktig input for Assigner
	outputAssigner := AssignerExecutable.Assigner(input)
	//konvertere outputAssigner til matriseform
	newOrderChannel <- MergeCABandHRAoutput(outputAssigner) //på formen Orders (ordermatrix)
}





//ID på CAB buttons 
//output fra assigner må appendes med CAB - MERGE
//MERGED må sendes til SINGLE ELEVATOR FSM
//må iterere gjennom keys og velge riktig elevator 
//velge riktig elevator og sette en på riktig sted til dens ordermatrix 
//MergeCABandHRA → Merges the converted HallOrderStatus with CabRequests to form a 4x3 matrix.
func MergeCABandHRAout(OutputAssigner map[string][][2]bool) single_elevator.Orders {
	var OrderMatrix single_elevator.Orders //initialiserer ordermatrix - fjerne initialisering i Single Elevator 
	for cabbuttons := range AssignerExecutable.HRAInput.CabRequests {
		if cabbuttons == true {
			OrderMatrix[cabbuttons][2] = true 
		}
	}
	//ikke riktig iterasjon??
	for ID, value := range OutputAssigner {
		if ID == WorldView.ID {
				for i, row := range orders {
					for j, value := range rows {
						if value == true {
							OrderMatrix[][] = true //hvilke elementer? 
						}
					}
				}
		}
	}
	return OrderMatrix
}

	//output fra assigner - map av id og hvilke ordre som skal tas 
	//legge til cab orders som en kolonne på høyre side 
	//alt må være bools og en 4x3 matrise 
	//return ordermatrix 






	func MergeWorldViews(localWorldView WorldView, updatedWorldView WorldView, IDsAliveElevators  []string) WorldView {
		//sjekke hvor mange som er i live??? hva skal vi gjøre med den infoen 
		//disse IDene må acknowledge og være i Acklist 

		if len(localWorldView.Acklist) < len(updatedWorldView.Acklist) {
			localWorldView = &updatedWorldView
			localWorldView.Acklist = append(localWorldView.ID)
		}
		if len(localWorldView.Acklist) = len(updatedWorldView.Acklist) {
			continue 
		} 
	}
//alle må ha oppdatert worldview før den kan assignes og utføres 

