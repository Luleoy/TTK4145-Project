package worldview

//unavailable state på single elevator - OPPDATERE Unavailable bool i Single Elevator
//lys

import (
	"TTK4145-Heislab/configuration"
	"TTK4145-Heislab/driver-go/elevio"
	"TTK4145-Heislab/single_elevator"

	"reflect"

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
}

func WorldViewManager(
	elevatorID string,
	WorldViewTXChannel chan<- WorldView,
	WorldViewRXChannel <-chan WorldView,
	buttonPressedChannel <-chan elevio.ButtonEvent,
	newOrderChannel chan<- single_elevator.Orders,
	completedOrderChannel <-chan elevio.ButtonEvent,
	IDPeersChannel <-chan []string,
	elevatorStateChannel <-chan single_elevator.Elevator,
) {

	initLocalWorldView := InitializeWorldView(elevatorID)
	localWorldView := &initLocalWorldView

	SendLocalWorldViewTimer := time.NewTimer(time.Duration(configuration.SendWVTimer))

	IDsAliveElevators := []string{elevatorID}

	var PreviousOrderMatrix single_elevator.Orders

	for {
		select {
		case IDList := <-IDPeersChannel:
			IDsAliveElevators = IDList //IDs alive is correct

		case <-SendLocalWorldViewTimer.C:
			//fmt.Println("Sending ww")
			localWorldView.ID = elevatorID
			WorldViewTXChannel <- localWorldView
			SetLights(localWorldView) //riktig oppdatering av lys?
			SendLocalWorldViewTimer.Reset(time.Duration(configuration.SendWVTimer))

		case elevatorState := <-elevatorStateChannel:
			elevStateMsg := localWorldView.ElevatorStatusList[elevatorID] // Hent en kopi av ElevStateMsg fra mappen
			elevStateMsg.Elev = elevatorState                             // Oppdater Elev-feltet i kopien
			localWorldView.ElevatorStatusList[elevatorID] = elevStateMsg  // Sett den oppdaterte structen tilbake i mappen
			//fmt.Println("floor: ", elevatorID, elevStateMsg.Elev.Floor)
			WorldViewTXChannel <- localWorldView // Send den oppdaterte WorldView til WorldViewTXChannel
			SetLights(localWorldView)            // Oppdater lysene

			//MÅ OPPDATERE HRAELEVSTATE

		case buttonPressed := <-buttonPressedChannel:
			newLocalWorldView := UpdateWorldViewWithButton(localWorldView, buttonPressed, true)
			if !ValidateWorldView(newLocalWorldView) {
				continue
			}
			localWorldView = &newLocalWorldView
			WorldViewTXChannel <- localWorldView
			SetLights(localWorldView)
			//denne er riktig
			//fmt.Println("Nå har vi oppdatert på TX kanalen. Har sendt LWV")

		case complete := <-completedOrderChannel:
			newLocalWorldView := UpdateWorldViewWithButton(localWorldView, complete, false)
			if !ValidateWorldView(newLocalWorldView) {
				continue
			}

			localWorldView = &newLocalWorldView
			WorldViewTXChannel <- localWorldView
			SetLights(localWorldView)

		//MESSAGE SYSTEM - connection with network
		case receivedWorldView := <-WorldViewRXChannel: //mottar en melding fra en annen heis
			//fmt.Println("Updated world view ", receivedWorldView.ElevatorStatusList[receivedWorldView.ID])
			newLocalWorldView := MergeWorldViews(localWorldView.deepcopy(), receivedWorldView, IDsAliveElevators)
			if !ValidateWorldView(newLocalWorldView) {
				continue
			}
			if !reflect.DeepEqual(newLocalWorldView, localWorldView) {
				//fmt.Println("WorldViews are different")
				WorldViewTXChannel <- newLocalWorldView
				localWorldView = newLocalWorldView
				SetLights(localWorldView)

				AssignHallOrders := AssignOrder(localWorldView, IDsAliveElevators)
				//fmt.Println("printing AsiignHallOrders: ", AssignHallOrders)

				OurHall := AssignHallOrders[localWorldView.ID]
				OurCab := GetOurCAB(localWorldView, localWorldView.ID)
				OrderMatrix := MergeCABandHRAout(OurHall, OurCab)
				if OrderMatrix != PreviousOrderMatrix {
					//fmt.Println("Fått en ny order")
					newOrderChannel <- OrderMatrix
					PreviousOrderMatrix = OrderMatrix
					// fmt.Println("ORDERMATRIX:" ,PreviousOrderMatrix)
				}
			}
		}
		SetLights(localWorldView)
		// WorldViewTXChannel <- l
	}
}
