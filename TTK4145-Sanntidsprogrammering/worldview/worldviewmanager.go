package worldview

//unavailable state på single elevator - OPPDATERE Unavailable bool i Single Elevator
//lys

import (
	"TTK4145-Heislab/configuration"
	"TTK4145-Heislab/driver-go/elevio"
	"TTK4145-Heislab/single_elevator"
	"fmt"
	"reflect"

	//"fmt"

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
	WorldViewTXChannel chan<- WorldView,
	WorldViewRXChannel <-chan WorldView,
	buttonPressedChannel <-chan elevio.ButtonEvent,
	newOrderChannel chan<- single_elevator.Orders,
	completedOrderChannel <-chan elevio.ButtonEvent,
	IDPeersChannel <-chan []string,
) {

	initLocalWorldView := InitializeWorldView(elevatorID)
	localWorldView := &initLocalWorldView

	SendLocalWorldViewTimer := time.NewTimer(time.Duration(configuration.SendWVTimer))

	IDsAliveElevators := []string{}

	var PreviousOrderMatrix single_elevator.Orders

	for {
		select {
		case IDList := <-IDPeersChannel:
			IDsAliveElevators = IDList //IDs alive is correct

		case <-SendLocalWorldViewTimer.C:
			fmt.Println("Sending ww")
			localWorldView.ID = elevatorID
			WorldViewTXChannel <- *localWorldView
			SetLights(*localWorldView) //riktig oppdatering av lys?
			SendLocalWorldViewTimer.Reset(time.Duration(configuration.SendWVTimer))

		case buttonPressed := <-buttonPressedChannel:
			newLocalWorldView := UpdateWorldViewWithButton(localWorldView, buttonPressed, true)
			if !ValidateWorldView(newLocalWorldView) {
				continue
			}
			localWorldView = &newLocalWorldView
			WorldViewTXChannel <- *localWorldView
			//denne er riktig
			//fmt.Println("Nå har vi oppdatert på TX kanalen. Har sendt LWV")

		case complete := <-completedOrderChannel:
			newLocalWorldView := UpdateWorldViewWithButton(localWorldView, complete, false)
			if !ValidateWorldView(newLocalWorldView) {
				continue
			}

			localWorldView = &newLocalWorldView
			WorldViewTXChannel <- *localWorldView

		//MESSAGE SYSTEM - connection with network
		case updatedWorldView := <-WorldViewRXChannel: //mottar en melding fra en annen heis
			fmt.Println("Got world view from: ", updatedWorldView.ID)
			//sammenligner counter for å avgjøre om meldingen skal brukes
			//oppdaterer localworldview hvis meldingen er nyere eller mer komplett
			//håndtering lys
			//oppdatere hallorderstatus basert på status for order
			//tildeler ordre hvis de ikke allerede er distribuert

			newLocalWorldView := MergeWorldViews(localWorldView, updatedWorldView, IDsAliveElevators)
			if !ValidateWorldView(newLocalWorldView) {
				continue
			}

			//HER MÅ VI GJØRE NOE SÅNN AT DET BARE SENDES EN NY WV PÅ TX HVIS DET ER ENDRIGER, SÅNN AT DET IKKE SENDES HELE TIDEN

			//if updatedWorldView.ID != elevatorID {
			// if newLocalWorldView != *localWorldView {
			// endre til å bare sende når newLocalWorldView != locaWorldView
			//WorldViewTXChannel <- newLocalWorldView
			//}

			if !reflect.DeepEqual(newLocalWorldView, *localWorldView) {
				fmt.Println("WorldViews are different")
				WorldViewTXChannel <- newLocalWorldView

			}

			AssignHallOrders := AssignOrder(*localWorldView, IDsAliveElevators)
			//fmt.Println("printing AsiignHallOrders: ", AssignHallOrders)

			OurHall := AssignHallOrders[localWorldView.ID]
			OurCab := GetOurCAB(*localWorldView, localWorldView.ID)
			OrderMatrix := MergeCABandHRAout(OurHall, OurCab)
			if OrderMatrix != PreviousOrderMatrix {
				//fmt.Println("Fått en ny order")
				newOrderChannel <- OrderMatrix
				PreviousOrderMatrix = OrderMatrix
			}
		}
	}
}

//lys
