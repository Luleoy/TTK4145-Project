package worldView

import (
	"TTK4145-Heislab/configuration"
	"TTK4145-Heislab/driver-go/elevio"
	"TTK4145-Heislab/singleElevator"
	"reflect"
	"time"
)

type ElevStateMsg struct {
	Elev singleElevator.Elevator
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
	newOrderChannel chan<- singleElevator.Orders,
	completedOrderChannel <-chan elevio.ButtonEvent,
	IDPeersChannel <-chan []string,
	elevatorStateChannel <-chan singleElevator.Elevator,
	elevatorTimeoutTimer *time.Timer,
) {

	initLocalWorldView := initializeWorldView(elevatorID)
	localWorldView := &initLocalWorldView

	SendLocalWorldViewTimer := time.NewTimer(time.Duration(configuration.SendWVTimer))

	IDsAliveElevators := []string{elevatorID}
	lastChanged := make(map[string]time.Time)
	lastChanged[elevatorID] = time.Now()

	var PreviousOrderMatrix singleElevator.Orders

	for {
		select {
		case IDList := <-IDPeersChannel:
			IDsAliveElevators = IDList

		case <-SendLocalWorldViewTimer.C: //Periodically broadcasts the elevators WorldView (every SendWVTimer) to synchronize elevator states across the network
			localWorldView.ID = elevatorID
			WorldViewTXChannel <- *localWorldView
			setLights(*localWorldView)
			SendLocalWorldViewTimer.Reset(time.Duration(configuration.SendWVTimer))

		case elevatorState := <-elevatorStateChannel: //Receives updated state (floor, direction, obstructed, behaviour) from SingleElevator and broadcasts this to synchronize across all elevators
			elevStateMsg := localWorldView.ElevatorStatusList[elevatorID]
			elevStateMsg.Elev = elevatorState
			localWorldView.ElevatorStatusList[elevatorID] = elevStateMsg
			WorldViewTXChannel <- *localWorldView
			setLights(*localWorldView)

		case buttonPressed := <-buttonPressedChannel:
			newLocalWorldView := updateWorldViewWithButton(localWorldView, buttonPressed, true)
			if !validateWorldView(newLocalWorldView) {
				continue
			}
			localWorldView = &newLocalWorldView
			WorldViewTXChannel <- *localWorldView
			setLights(*localWorldView)

		case complete := <-completedOrderChannel: //Receives order completion events from SingleElevator and updates WorldView
			newLocalWorldView := updateWorldViewWithButton(localWorldView, complete, false)
			if !validateWorldView(newLocalWorldView) {
				continue
			}
			localWorldView = &newLocalWorldView
			WorldViewTXChannel <- *localWorldView
			setLights(*localWorldView)

		case receivedWorldView := <-WorldViewRXChannel:
			lastChanged = updateLastChanged(*localWorldView, receivedWorldView, lastChanged)
			IDsAvailableForAssignment := []string{elevatorID}
			for _, id := range IDsAliveElevators {
				if lastChange_i, ok := lastChanged[id]; ok && id != elevatorID {
					if time.Now().Sub(lastChange_i) < 10*time.Second {
						IDsAvailableForAssignment = append(IDsAvailableForAssignment, id)
					}
				}
			}
			newLocalWorldView := mergeWorldViews(localWorldView, receivedWorldView, IDsAvailableForAssignment)
			if !validateWorldView(newLocalWorldView) {
				continue
			}
			if !reflect.DeepEqual(newLocalWorldView, *localWorldView) {
				WorldViewTXChannel <- newLocalWorldView
				localWorldView = &newLocalWorldView
				setLights(*localWorldView)
			}
			AssignHallOrders := assignOrder(*localWorldView, IDsAvailableForAssignment)
			OurHall := AssignHallOrders[localWorldView.ID]
			OurCab := getCAB(*localWorldView, localWorldView.ID)
			OrderMatrix := mergeCABandHRAOutput(OurHall, OurCab)
			if OrderMatrix != PreviousOrderMatrix {
				newOrderChannel <- OrderMatrix
				PreviousOrderMatrix = OrderMatrix
				anyOrders := false
				for i := 0; i < configuration.NumFloors; i++ {
					for j := 0; j < configuration.NumButtons; j++ {
						anyOrders = anyOrders || OrderMatrix[i][j]
					}
				}
			}
		}
		setLights(*localWorldView)
	}
}
