package single_elevator

import (
	"TTK4145-Heislab/configuration"
	"TTK4145-Heislab/driver-go/elevio"
	"fmt"
	"time"
)

type Elevator struct { //the elevators current state
	Floor       int
	Direction   Direction //directions: Up, Down
	Obstructed  bool
	Behaviour   Behaviour //behaviours: Idle, Moving and DoorOpen
	Unavailable bool      //MÅ OPPDATERE
}

type Behaviour int

const (
	Idle Behaviour = iota
	Moving
	DoorOpen
)

func ToString(behaviour Behaviour) string {
	switch behaviour {
	case Idle:
		return "Idle"
	case Moving:
		return "Moving"
	case DoorOpen:
		return "DoorOpen"
	default:
		return "Unknown"
	}
}

func runTimer(duration time.Duration, timeOutChannel chan<- bool, resetTimerChannel <-chan bool) {
	deadline := time.Now().Add(100000 * time.Hour)
	is_running := false

	for {
		select {
		case <-resetTimerChannel:
			deadline = time.Now().Add(duration)
			is_running = true
		default:
			if is_running && time.Since(deadline) > 0 {
				timeOutChannel <- true
				is_running = false
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func SingleElevator(
	newOrderChannel <-chan Orders, //receiving new orders FROM ORDER MANAGER
	completedOrderChannel chan<- elevio.ButtonEvent, //sending information about completed orders TO ORDER MANAGER
	newLocalStateChannel chan<- Elevator, //sending information about the elevators current state TO ORDER MANAGER
) {
	//Initialization of elevator
	fmt.Println("setting motor down")

	//elevio.SetMotorDirection(elevio.MD_Down)
	//state := Elevator{Direction: Down, Behaviour: Moving}
	var state Elevator
	currentFloor := elevio.GetFloor()

	if currentFloor != -1 {
		fmt.Println("Heis starter i etasje", currentFloor)
		state = Elevator{Floor: currentFloor, Behaviour: Idle, Direction: elevio.MD_Stop}
	} else {
		fmt.Println("Elevator beetween floors, go down")
		elevio.SetMotorDirection(elevio.MD_Down)
		closestFloor := findClosestFloor()
		fmt.Println("floor found: ", closestFloor)
		elevio.SetMotorDirection(elevio.MD_Stop)
		state = Elevator{Floor: closestFloor, Behaviour: Idle, Direction: elevio.MD_Stop}
	}

	floorEnteredChannel := make(chan int)
	obstructedChannel := make(chan bool, 16)
	stopPressedChannel := make(chan bool, 16)

	go elevio.PollFloorSensor(floorEnteredChannel)

	timerOutChannel := make(chan bool)
	resetTimerChannel := make(chan bool)
	go runTimer(configuration.DoorOpenDuration, timerOutChannel, resetTimerChannel)
	// go startTimer(configuration.DoorOpenDuration, timerOutChannel)

	go elevio.PollObstructionSwitch(obstructedChannel)
	go elevio.PollStopButton(stopPressedChannel)

	var OrderMatrix Orders

	for {
		select {
		case <-timerOutChannel:
			switch state.Behaviour {
			case DoorOpen:
				DirectionBehaviourPair := ordersChooseDirection(state.Floor, state.Direction, OrderMatrix)
				state.Behaviour = DirectionBehaviourPair.Behaviour
				state.Direction = Direction(DirectionBehaviourPair.Direction)
				newLocalStateChannel <- state
				switch state.Behaviour {
				case DoorOpen:
					resetTimerChannel <- true
					OrderCompletedatCurrentFloor(state.Floor, Direction(state.Direction.convertMD()), completedOrderChannel)
					newLocalStateChannel <- state
				case Moving, Idle:
					elevio.SetDoorOpenLamp(false)
					elevio.SetMotorDirection(DirectionBehaviourPair.Direction)

				}
			case Moving:
				panic("timeroutchannel in moving")
			}
		case stopbuttonpressed := <-stopPressedChannel:
			if stopbuttonpressed {
				fmt.Println("StopButton is pressed")
				elevio.SetStopLamp(true)
				elevio.SetMotorDirection(elevio.MD_Stop)
			} else {
				elevio.SetStopLamp(false)
			}
		case obstruction := <-obstructedChannel:
			if obstruction {
				state.Obstructed = true
				state.Unavailable = true
				fmt.Println("Obstruction detected! Elevator unavailable")
				state.Behaviour = DoorOpen
				elevio.SetDoorOpenLamp(true)
				newLocalStateChannel <- state
				resetTimerChannel <- true
				for obstruction {
					select {
					case obstruction = <-obstructedChannel:
						if !obstruction {
							state.Obstructed = false
							state.Unavailable = false
							fmt.Println("Obstruction cleared! Elevator available.")
							newLocalStateChannel <- state
							if state.Behaviour == DoorOpen {
								resetTimerChannel <- true
							}
						}
					default:
						if state.Behaviour == DoorOpen {
							resetTimerChannel <- true
						}
					}
				}
			}
		case state.Floor = <-floorEnteredChannel:
			fmt.Println("New floor: ", state.Floor)
			elevio.SetFloorIndicator(state.Floor)
			switch state.Behaviour {
			case Moving:
				if orderHere(OrderMatrix, state.Floor) || state.Floor == 0 || state.Floor == configuration.NumFloors-1 {
					elevio.SetMotorDirection(elevio.MD_Stop)
					OrderCompletedatCurrentFloor(state.Floor, Direction(state.Direction.convertMD()), completedOrderChannel)
					resetTimerChannel <- true
					state.Behaviour = DoorOpen
					newLocalStateChannel <- state
					fmt.Println("New local state:", state)
				}
			default:
			}
		case OrderMatrix = <-newOrderChannel:
			fmt.Println("New orders :)")
			switch state.Behaviour {
			case Idle:
				state.Behaviour = Moving
				DirectionBehaviourPair := ordersChooseDirection(state.Floor, state.Direction, OrderMatrix)
				state.Behaviour = DirectionBehaviourPair.Behaviour
				state.Direction = Direction(DirectionBehaviourPair.Direction)
				newLocalStateChannel <- state
				switch state.Behaviour {
				case DoorOpen:
					resetTimerChannel <- true
					OrderCompletedatCurrentFloor(state.Floor, Direction(state.Direction.convertMD()), completedOrderChannel)
					newLocalStateChannel <- state
				case Moving, Idle:
					elevio.SetDoorOpenLamp(false)
					elevio.SetMotorDirection(DirectionBehaviourPair.Direction)
				}
			}
		}
		elevio.SetDoorOpenLamp(state.Behaviour == DoorOpen)
	}
}

/*
where to update when elevator is unavailable?
initialization of elevator. go down to nearest floor.- hva har andre gjort?
default/panic bør det implementeres over alt?
doesnt know its in between two floors when stopping in between two floors
printer new orders selv om vi ikke har noen orders?
fjerne newlocalstate - brukes ikke noe sted
*/
