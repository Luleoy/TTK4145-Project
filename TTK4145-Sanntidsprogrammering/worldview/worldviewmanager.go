package worldview

//håndtering av CAB requests i WorldView sånn at vi ikke trenger å save to file - CAB reuests when converting to HRAInput
//konvertering fra Worldstate til HRA til Single Elevator
//unavailable state på single elevator - OPPDATERE Unavailable bool i Single Elevator
//assignedOrdersChannel vs newOrderChannel - sender OrderMatrix på newOrderChannel. må konvertere

//FSM - single elevator konvertering og kommunikasjon
//neworderchannel og completedorderchannel
//buttonpressedchannel??

//test

import (
	"TTK4145-Heislab/configuration"
	"TTK4145-Heislab/driver-go/elevio"
	"TTK4145-Heislab/single_elevator"
	"time"
)

// REQUESTSTATE INT - lage en funksjon som konverterer fra WorldView til OrderMatrix som vi kan bruke
// vi vil ha noe spesifikt å sende til FSM
type ElevStateMsg struct {
	Elev single_elevator.State
	Cab  []configuration.OrderMsg
}

// oppdaterer newlocalstate i single_elevator FSM
type WorldView struct {
	ID string
	// Acklist []string

	ElevatorStatusList map[string]ElevStateMsg //legger inn al			hvordan fysisk kobler man opp....
	// l info om hver heis; floor, direction, obstructed, behaviour
	HallOrderStatus [][configuration.NumButtons - 1]configuration.OrderMsg
	// CabRequests        [configuration.NumFloors]configuration.OrderMsg //hvis vi broadcaster cab buttons her, slipper vi å lagre alt over i en fil
	//Burde vi broadcaste om heisen er i live eller ikke/unavailable??
}

func WorldViewManager(
	elevatorID string,
	WorldViewTXChannel chan<- WorldView, //WorldView transmitter
	WorldViewRXChannel <-chan WorldView, //WorldView receiver
	buttonPressedChannel <-chan elevio.ButtonEvent,
	mergeChannel chan<- elevio.ButtonEvent,
	newOrderChannel chan<- single_elevator.Orders, //skal brukes i single elevator - fra OrderManager
	completedOrderChannel <-chan elevio.ButtonEvent,
	numPeersChannel <-chan int,
	IDPeersChannel <-chan []string,
) {

	//initialize local world view to send on message channel
	initLocalWorldView := InitializeWorldView(elevatorID)
	localWorldView := &initLocalWorldView //bruke localworldview i casene fremover - DYP kopiere worldview

	//timer for når Local World View skal oppdateres
	SendLocalWorldViewTimer := time.NewTimer(time.Duration(configuration.SendWVTimer) * time.Millisecond)
	numPeers := 0 //må denne initialiseres?

	orderDistributed := make([][configuration.NumButtons - 1]bool, configuration.NumFloors) //liste for at alle heisene skal vite at de har distribuert order

	IDsAliveElevators := []string{}

	for {
	OuterLoop: //break ut av hele for-loopen
		select {
		case IDList := <-IDPeersChannel:
			numPeers = len(IDList)
			IDsAliveElevators = IDList

		case <-SendLocalWorldViewTimer.C: //local world view updates
			localWorldView.ID = elevatorID
			WorldViewTXChannel <- *localWorldView
			SendLocalWorldViewTimer.Reset(time.Duration(configuration.SendWVTimer) * time.Millisecond)

		case buttonPressed := <-buttonPressedChannel: //knappetrykk. tar inn button events. Dette er neworder. Må skille fra Neworderchannel i single_elevator. sjekk ut hvor den skal defineres etc
			newLocalWorldView := updateWorldViewWithButton(localWorldView, buttonPressed, true) // false if remove this order
			//feilhåndtering
			if !validWorldView(newLocalWorldView) { //ikke laget validWorldView enda
				continue
			}
			localWorldView = &newLocalWorldView
			WorldViewTXChannel <- *localWorldView //la til peker?

		case complete := <-completedOrderChannel:
			newLocalWorldView := updateWorldViewWithButton(localWorldView, complete, false) // false if remove this order
			//feilhåndtering
			if !validWorldView(newLocalWorldView) { //ikke laget validWorldView enda
				continue
			}
			localWorldView = &newLocalWorldView
			WorldViewTXChannel <- *localWorldView //la til peker?

		//MESSAGE SYSTEM - connection with network
		case updatedWorldView := <-WorldViewRXChannel: //mottar en melding fra en annen heis
			//sammenligner counter for å avgjøre om meldingen skal brukes
			//oppdaterer localworldview hvis meldingen er nyere eller mer komplett
			//håndtering lys
			//oppdatere hallorderstatus basert på status for order
			//tildeler ordre hvis de ikke allerede er distribuert

			newLocalWorldView = MergeWorldViews(localWorldView, updatedWorldView, IDsAliveElevators)
			if !validWorldView(newLocalWorldView) {
				continue
			}
			// send new worldview on network

			//UPDATE HALLSTATUS TO CONFIRMED
			//ackliste bare skal være så lang som aktive heiser
			//SJEKKE OM ACKLIST ER LIKE LANG SOM AKTIVE HEISER - samme IDer

			AssignHallOrders := AssignOrder(*localWorldView)
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

			//assign order
			// run HRA and send new ordermatrix to single elev
		}
	}
}

//lys
