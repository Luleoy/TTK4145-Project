package worldview

//unavailable state på single elevator - OPPDATERE Unavailable bool i Single Elevator
//lys

import (
	"TTK4145-Heislab/configuration"
	"TTK4145-Heislab/driver-go/elevio"
	"TTK4145-Heislab/single_elevator"
	"fmt"

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
	elevatorTimeoutTimer *time.Timer,
) {

	initLocalWorldView := InitializeWorldView(elevatorID)
	localWorldView := &initLocalWorldView

	SendLocalWorldViewTimer := time.NewTimer(time.Duration(configuration.SendWVTimer))

	IDsAliveElevators := []string{elevatorID}
	lastChanged := make(map[string]time.Time) // For å holde styr på siste gang vi så hver heis
	lastChanged[elevatorID] = time.Now()      // Initialiser for vår egen heis

	var PreviousOrderMatrix single_elevator.Orders

	for {
		select {
		case IDList := <-IDPeersChannel:
			IDsAliveElevators = IDList
			// now := time.Now()
			// Update last seen for all received IDs
			// for _, id := range IDList {
			// 	lastSeen[id] = now
			// }

		//case <-elevatorTimeoutTimer.C:
		// now := time.Now()
		// newAliveList := []string{}
		// for id, elevState := range localWorldView.ElevatorStatusList {
		// 	if now.Sub(lastSeen[id]) < 5*time.Second || elevState.Elev.Behaviour == single_elevator.Idle {
		// 		newAliveList = append(newAliveList, id)
		// 	}
		// }
		// IDsAliveElevators = newAliveList
		// elevatorTimeoutTimer.Reset(5 * time.Second)

		case <-SendLocalWorldViewTimer.C:
			//fmt.Println("Sending ww")
			localWorldView.ID = elevatorID
			WorldViewTXChannel <- *localWorldView
			SetLights(*localWorldView) //riktig oppdatering av lys?
			SendLocalWorldViewTimer.Reset(time.Duration(configuration.SendWVTimer))

		case elevatorState := <-elevatorStateChannel:
			elevStateMsg := localWorldView.ElevatorStatusList[elevatorID] // Hent en kopi av ElevStateMsg fra mappen
			elevStateMsg.Elev = elevatorState                             // Oppdater Elev-feltet i kopien
			localWorldView.ElevatorStatusList[elevatorID] = elevStateMsg  // Sett den oppdaterte structen tilbake i mappen

			//fmt.Println("floor: ", elevatorID, elevStateMsg.Elev.Floor)
			WorldViewTXChannel <- *localWorldView // Send den oppdaterte WorldView til WorldViewTXChannel
			SetLights(*localWorldView)            // Oppdater lysene

		case buttonPressed := <-buttonPressedChannel:
			newLocalWorldView := UpdateWorldViewWithButton(localWorldView, buttonPressed, true)
			if !ValidateWorldView(newLocalWorldView) {
				continue
			}
			localWorldView = &newLocalWorldView
			WorldViewTXChannel <- *localWorldView
			SetLights(*localWorldView)
			//denne er riktig
			//fmt.Println("Nå har vi oppdatert på TX kanalen. Har sendt LWV")

		case complete := <-completedOrderChannel:
			newLocalWorldView := UpdateWorldViewWithButton(localWorldView, complete, false)
			if !ValidateWorldView(newLocalWorldView) {
				continue
			}

			localWorldView = &newLocalWorldView
			WorldViewTXChannel <- *localWorldView
			SetLights(*localWorldView)

		//MESSAGE SYSTEM - connection with network
		case receivedWorldView := <-WorldViewRXChannel: //mottar en melding fra en annen heis

			lastChanged = UpdateLastChanged(*localWorldView, receivedWorldView, lastChanged)

			IDsAvailableForAssignment := []string{elevatorID}
			for _, id := range IDsAliveElevators {
				if lastChange_i, ok := lastChanged[id]; ok && id != elevatorID {
					if time.Now().Sub(lastChange_i) < 10*time.Second {
						IDsAvailableForAssignment = append(IDsAvailableForAssignment, id)
					}
				}
			}
			// if len(IDsAvailableForAssignment) == 0{
			// 	fmt.Println("Only us which can take order??")
			// } else {
			// 	fmt.Println("Assigment available: ", IDsAvailableForAssignment)
			// }

			// lastChanged = updateLastChanged(localWorldView, receivedWorldView, lastChanged)
			///
			// id == recievedww.eleavtorId
			// if "id not in localww" || locaworldview[id].state != recievedww[id].state || localww[id].state == Idle  {
			//	lastChanged[id] = time.Now()
			//}
			//

			// idsAvailableForAssignment = []...
			// for id in idsalive {
			// if lastChanged[id].sub(...) > 0 {
			// idsAliveForAssigment[id] = true
			//}
			//}

			//fmt.Println("Updated world view ", receivedWorldView.ElevatorStatusList[receivedWorldView.ID])
			newLocalWorldView := MergeWorldViews(localWorldView, receivedWorldView, IDsAvailableForAssignment)
			if !ValidateWorldView(newLocalWorldView) {
				continue
			}
			if !reflect.DeepEqual(newLocalWorldView, *localWorldView) {
				//fmt.Println("WorldViews are different")
				WorldViewTXChannel <- newLocalWorldView
				localWorldView = &newLocalWorldView
				SetLights(*localWorldView)
			}
			AssignHallOrders := AssignOrder(*localWorldView, IDsAvailableForAssignment)

			// fmt.Println("\n\nprinting AsiignHallOrders: ", AssignHallOrders)

			OurHall := AssignHallOrders[localWorldView.ID]
			OurCab := GetOurCAB(*localWorldView, localWorldView.ID)
			OrderMatrix := MergeCABandHRAout(OurHall, OurCab)
			if OrderMatrix != PreviousOrderMatrix {
				//fmt.Println("Fått en ny order")
				newOrderChannel <- OrderMatrix
				PreviousOrderMatrix = OrderMatrix
				//fmt.Println("ORDERMATRIX:", PreviousOrderMatrix)
				anyOrders := false
				for i := 0; i < configuration.NumFloors; i++ {
					for j := 0; j < configuration.NumButtons; j++ {
						anyOrders = anyOrders || OrderMatrix[i][j]
					}
				}
				if !anyOrders {
					fmt.Println("No orders")
				} else {
					fmt.Println("My orders: ", OrderMatrix)
				}
				fmt.Println("A state: ", localWorldView.ElevatorStatusList["A"])
				fmt.Println("C state: ", localWorldView.ElevatorStatusList["C"])
			}

			// panic("test")
		}
		SetLights(*localWorldView)
		// WorldViewTXChannel <- l
	}
}
