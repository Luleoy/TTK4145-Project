package worldview

//TRENGER FRA COMMUNICATION: numPeers

import (
	"TTK4145-Heislab/AssignerExecutable"
	"TTK4145-Heislab/configuration"
	"TTK4145-Heislab/driver-go/elevio"
	"TTK4145-Heislab/single_elevator"
)

// REQUESTSTATE INT - lage en funksjon som konverterer fra WorldView til OrderMatrix som vi kan bruke
//vi vil ha noe spesifikt å sende til FSM

func InitializeWorldView(elevatorID string) WorldView {
	message := WorldView{
		Counter:            0,
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

//gjør to ting: samler info om alle aktive heiser i systemet (elevatorstatuslist in WORLDVIEW), henter en oversikt over hall-bestillinger fra WORLDVIEW

//func Assigner(input HRAInput) map[string][][2]bool
/*
opprette tom map for heistilstander (elevstates) - info om hver heis som er aktiv i dette systemet (elevatorstatuslist)
returnerer HRAINPUT
hente alle hallbestillinger fra worldview
gå gjennom acklist for å finne ut av hvilke heiser som er aktive
for hver heis i acklist, hvis heisen ikke er utligjengelig, legges den til elevstates og info om heisen hentes i elevtor list i worldview
henter behaviour, floor, direction (cabrequests)
opprette HRAInput struktur og returnere - sette inn hallrequests og states
*/

//type HRAElevState struct {
//Behavior    string `json:"behaviour"`
//Floor       int    `json:"floor"`
//Direction   string `json:"direction"`
//CabRequests []bool `json:"cabRequests"`
//}

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
				//CABREQUESTS - hvordan håndtere
			}
		}
	}
	input := AssignerExecutable.HRAInput{
		HallRequests: hallrequests,
		States:       elevatorStates,
	}
	return input
}

//konvertere HallOrderStatus sånn ta den er bool
//ta inn bool matrise inn i assginer
//output fra assigner må appendes med CAB - MERGE
//MERGED må sendes til SINGLE ELEVATOR FSM

// assignedOrdersChannel vs newOrderChannel - sender OrderMatrix på newOrderChannel
func AssignOrder(WorldView WorldView, assignedOrdersChannel chan<- map[string][][2]bool) { //map med ID som nøkkel, og arrays med 2 bolske verdier med orders true or false
	input := HRAInputFormatting(WorldView) //Konverterer WorldView til riktig input for Assigner
	assignedOrdersChannel <- AssignerExecutable.Assigner(input)
}

//prøve å snakke med FSM
/*
ConvertHallOrderStatustoBool → Converts HallOrderStatus to a boolean matrix.
MergeCABandHRA → Merges the converted HallOrderStatus with CabRequests to form a 4x3 matrix.

ta inn map struktur fordi den kommer fra assigner

func MergeCABandHRA(HallOrderStatus [][]bool, CabRequests []bool) [][]bool {
	// Ensure both inputs have the same number of floors
	if len(HallOrderStatus) != len(CabRequests) {
		panic("Dimension mismatch: HallOrderStatus and CabRequests must have the same number of floors")
	}

	// Create a new 4x3 matrix
	OrderMatrix := make([][]bool, len(HallOrderStatus))
	for i := range HallOrderStatus {
		OrderMatrix[i] = append(HallOrderStatus[i], CabRequests[i]) // Append CabRequest[i] as a new column
	}

	return OrderMatrix
}
*/
