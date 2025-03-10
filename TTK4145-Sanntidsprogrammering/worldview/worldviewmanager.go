package worldview

//unavailable state på single elevator - OPPDATERE Unavailable bool i Single Elevator
//lys

import (
	"TTK4145-Heislab/configuration"
	"TTK4145-Heislab/driver-go/elevio"
	"TTK4145-Heislab/single_elevator"
	"time"
)

type ElevStateMsg struct {
	Elev single_elevator.Elevator
	Cab  []configuration.OrderMsg
}

type WorldView struct {
	ID                 string
	ElevatorStatusList map[string]ElevStateMsg
	HallOrderStatus    [][configuration.NumButtons - 1]configuration.OrderMsg
	//Burde vi broadcaste om heisen er unavailable??
}

func WorldViewManager(
	elevatorID string,
	WorldViewTXChannel chan<- WorldView, //WorldView transmitter
	WorldViewRXChannel <-chan WorldView, //WorldView receiver
	buttonPressedChannel <-chan elevio.ButtonEvent,
	mergeChannel chan<- elevio.ButtonEvent,
	newOrderChannel chan<- single_elevator.Orders,
	completedOrderChannel <-chan elevio.ButtonEvent,
	numPeersChannel <-chan int,
	IDPeersChannel <-chan []string,
) {

	initLocalWorldView := InitializeWorldView(elevatorID)
	localWorldView := &initLocalWorldView

	SendLocalWorldViewTimer := time.NewTimer(time.Duration(configuration.SendWVTimer) * time.Millisecond)

	IDsAliveElevators := []string{}

	for {
		select {
		case IDList := <-IDPeersChannel:
			IDsAliveElevators = IDList

		case <-SendLocalWorldViewTimer.C:
			localWorldView.ID = elevatorID
			WorldViewTXChannel <- *localWorldView
			SendLocalWorldViewTimer.Reset(time.Duration(configuration.SendWVTimer) * time.Millisecond)

		case buttonPressed := <-buttonPressedChannel:
			newLocalWorldView := updateWorldViewWithButton(localWorldView, buttonPressed, true)
			if !ValidateWorldView(newLocalWorldView) {
				continue
			}
			localWorldView = &newLocalWorldView
			WorldViewTXChannel <- *localWorldView

		case complete := <-completedOrderChannel:
			newLocalWorldView := updateWorldViewWithButton(localWorldView, complete, false)
			if !ValidateWorldView(newLocalWorldView) {
				continue
			}

			localWorldView = &newLocalWorldView
			WorldViewTXChannel <- *localWorldView

		//MESSAGE SYSTEM - connection with network
		case updatedWorldView := <-WorldViewRXChannel: //mottar en melding fra en annen heis
			//sammenligner counter for å avgjøre om meldingen skal brukes
			//oppdaterer localworldview hvis meldingen er nyere eller mer komplett
			//håndtering lys
			//oppdatere hallorderstatus basert på status for order
			//tildeler ordre hvis de ikke allerede er distribuert

			newLocalWorldView := MergeWorldViews(*localWorldView, updatedWorldView, IDsAliveElevators)
			if !ValidateWorldView(newLocalWorldView) { //ikke laget validWorldView enda
				continue
			}
			WorldViewTXChannel <- newLocalWorldView
			// send new worldview on network - må gjøre noe med mergeworldviews?

			//UPDATE HALLSTATUS TO CONFIRMED
			//ackliste bare skal være så lang som aktive heiser
			//SJEKKE OM ACKLIST ER LIKE LANG SOM AKTIVE HEISER - samme IDer

			AssignHallOrders := AssignOrder(*localWorldView, IDsAliveElevators)
			OurHall := AssignHallOrders[localWorldView.ID] //value ut av map
			OurCab := GetOurCAB(*localWorldView, localWorldView.ID)
			OrderMatrix := MergeCABandHRAout(OurHall, OurCab)
			newOrderChannel <- OrderMatrix

			/*
				hraInput := convertToHra(localWorldView)
				assignedHalorders := runHRA(hraInput)
				ourHal := assignedHalorders[ourId]
				ourCab := getOurCab(localWorldView, ourId)
				ordermatrix := covertToOrderMatrix(ourHal, ourCab)
				updatedOrdersChan <- ordermatrix
			*/
		}
	}
}

//lys
//packetloss - håndterer vel egt dette?
