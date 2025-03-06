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
	"reflect"
	"time"
)

// REQUESTSTATE INT - lage en funksjon som konverterer fra WorldView til OrderMatrix som vi kan bruke
//vi vil ha noe spesifikt å sende til FSM

// oppdaterer newlocalstate i single_elevator FSM
type WorldView struct {
	Counter int
	ID      string
	Acklist []string

	ElevatorStatusList map[string]single_elevator.State //legger inn all info om hver heis; floor, direction, obstructed, behaviour
	HallOrderStatus    [][configuration.NumButtons - 1]configuration.RequestState
	CabRequests        [configuration.NumFloors]bool //hvis vi broadcaster cab buttons her, slipper vi å lagre alt over i en fil
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
) {

	//initialize local world view to send on message channel
	initLocalWorldView := InitializeWorldView(elevatorID)
	localWorldView := &initLocalWorldView //bruke localworldview i casene fremover - DYP kopiere worldview

	//timer for når Local World View skal oppdateres
	SendLocalWorldViewTimer := time.NewTimer(time.Duration(configuration.SendWVTimer) * time.Millisecond)
	numPeers := 0 //må denne initialiseres?

	orderDistributed := make([][configuration.NumButtons - 1]bool, configuration.NumFloors) //liste for at alle heisene skal vite at de har distribuert order

	for {
	OuterLoop: //break ut av hele for-loopen
		select {
		case num := <-numPeersChannel:
			numPeers = num

		case <-SendLocalWorldViewTimer.C: //local world view updates
			localWorldView.ID = elevatorID
			WorldViewTXChannel <- *localWorldView
			SendLocalWorldViewTimer.Reset(time.Duration(configuration.SendWVTimer) * time.Millisecond)

		case buttonPressed := <-buttonPressedChannel: //knappetrykk. tar inn button events. Dette er neworder. Må skille fra Neworderchannel i single_elevator. sjekk ut hvor den skal defineres etc
			//1 heis - sende til SINGLE ELEVATOR
			if numPeers == 1 { //kun en heis - har tatt inn ordermanager funksjonen
				OrderMatrix := [configuration.NumFloors][configuration.NumButtons]bool{}
				OrderMatrix[buttonPressed.Floor][buttonPressed.Button] = true
				//SetLights(OrderMatrix)
				newOrderChannel <- OrderMatrix
				//hvordan skal vi complete order når det er en single elevator? - skal det sendes til completeorderchannel?
				/*for {
					select {
					case ordercompletedbyfsm := <-completedOrderChannel: //KAN IKKE LESE UT AV COMPLETEDORDERCHANNEL FLERE STEDER
						OrderMatrix[ordercompletedbyfsm.Floor][ordercompletedbyfsm.Button] = false
						//SetLights(OrderMatrix)
						newOrderChannel <- OrderMatrix
					}
				} */
			}
			//CAB BUTTONS - sende til SINGLE ELEVATOR
			if buttonPressed.Button == elevio.BT_Cab {
				//oppdatere worldview med CAB buttons
				//SENERE: merge CAB buttons og eksisterende HALL buttons inn i en OrderMatrix - vil ha den oppdaterte matrisen OG SEND TIL SINGLE ELEVATOR
			}
			localWorldView.HallOrderStatus[buttonPressed.Floor][int(buttonPressed.Button)] = configuration.Order //setter hallorder i hallorderstatus til ORDER
			localWorldView.Counter++                                                                             //øker counter
			ResetAckList(localWorldView)                                                                         //tømmer ackliste og legger til egen ID

		//MESSAGE SYSTEM - connection with network
		case updatedWorldView := <-WorldViewRXChannel: //mottar en melding fra en annen heis
			//sammenligner counter for å avgjøre om meldingen skal brukes
			//oppdaterer localworldview hvis meldingen er nyere eller mer komplett
			//håndtering lys
			//oppdatere hallorderstatus basert på status for order
			//tildeler ordre hvis de ikke allerede er distribuert

			if localWorldView.Counter >= updatedWorldView.Counter { //sjekker lengde av egen counter og counter på melding
				if localWorldView.Counter == updatedWorldView.Counter && len(localWorldView.Acklist) < len(updatedWorldView.Acklist) { //hvis counters er like, og acklisten til melding er lengre
					localElevatorStatus := localWorldView.ElevatorStatusList[elevatorID] //henter status fra elevatorID
					localWorldView = &updatedWorldView                                   //update egen world view
					localWorldView.ElevatorStatusList[elevatorID] = localElevatorStatus
				} else {
					break OuterLoop
				}
				//set lights?
				//håndtere heisbestillinger
				if len(updatedWorldView.Acklist) == numPeers { //alle heiser har acknowledged og lagt seg til i acklist
					for floor := 0; floor < configuration.NumFloors; floor++ { //iterere gjennom floors
						for button := 0; button < configuration.NumButtons-1; button++ { //iterere gjennom buttons
							switch {
							case updatedWorldView.HallOrderStatus[floor][button] == configuration.Order: //legger til hallorder
								localWorldView.HallOrderStatus[floor][button] = configuration.Confirmed //confirmed hallorder
								localWorldView.Counter = updatedWorldView.Counter                       //setter counter lik hverandre
								localWorldView.Counter++                                                //øker counter
								ResetAckList(localWorldView)                                            //resetter acklist og legger seg selv til i acklist
							case updatedWorldView.HallOrderStatus[floor][button] == configuration.Confirmed && !orderDistributed[floor][button]:
								CompleteOrder()
								AssignOrder(updatedWorldView, newOrderChannel)
								orderDistributed[floor][button] = true
								localWorldView = &updatedWorldView
								localWorldView.ID = elevatorID
								//bestillinger i Confirmed-status som ikke er distribuert, tildeles en heis - AssignOrder
								//case må fylles inn
							case updatedWorldView.HallOrderStatus[floor][button] == configuration.Complete:
								localWorldView.HallOrderStatus[floor][button] = configuration.None
								orderDistributed[floor][button] = false
								localWorldView.Counter++
							}
						}
					}
				} else {
					for IDs := range updatedWorldView.Acklist {
						if localWorldView.ID == updatedWorldView.Acklist[IDs] {
							if reflect.DeepEqual(localWorldView.Acklist, updatedWorldView.Acklist) {
								localElevatorStatus := localWorldView.ElevatorStatusList[elevatorID] //henter status fra elevatorID
								localWorldView = &updatedWorldView                                   //update egen world view
								localWorldView.ElevatorStatusList[elevatorID] = localElevatorStatus

								break OuterLoop
							}

							localElevatorStatus := localWorldView.ElevatorStatusList[elevatorID] //henter status fra elevatorID
							localWorldView = &updatedWorldView                                   //update egen world view
							localWorldView.ElevatorStatusList[elevatorID] = localElevatorStatus

							localWorldView.Counter++
							break OuterLoop
						}
					}
					localElevatorStatus := localWorldView.ElevatorStatusList[elevatorID]
					localWorldView = &updatedWorldView
					localWorldView.ElevatorStatusList[elevatorID] = localElevatorStatus

					localWorldView.Acklist = append(localWorldView.Acklist, elevatorID)
					localWorldView.Counter++

					if len(updatedWorldView.Acklist) == numPeers {
						for floor := 0; floor < configuration.NumFloors; floor++ {
							for button := 0; button < configuration.NumButtons-1; button++ {
								if localWorldView.HallOrderStatus[floor][button] == configuration.Confirmed && !orderDistributed[floor][button] {
									CompleteOrder(localWorldView, button, floor, true)
									AssignOrder(updatedWorldView, newOrderChannel)
									orderDistributed[floor][button] = true
								}
							}
						}
					}
				}
			}
			//? sammenheng med single elevator
			//case complete := <-completedOrderChannel: //må få inn complete order fra FSM - når single elevator også?
			//motta bekreftelse på at ordre er fullført
			//oppdatere hallorderstatus til complete
			//lights
			//øke counter

			//order completed SINGLE ELEVATOR FSM:
			//case ordercompletedbyfsm := <-completedOrderChannel: completed order channel sender button event
			//OrderMatrix[ordercompletedbyfsm.Floor][ordercompletedbyfsm.Button] = false
			//SetLights(OrderMatrix)
			//newOrderChannel <- OrderMatrix
		}
	}
}

//MÅ LAGE HALLORDERDISTRIBUTOR
//lys
