package singleElevator

import (
	"TTK4145-Heislab/configuration"
	"TTK4145-Heislab/driver-go/elevio"
	"time"
)

type Elevator struct {
	Floor      int
	Direction  Direction
	Obstructed bool
	Behaviour  Behaviour
}

type Behaviour int

const (
	Idle Behaviour = iota
	Moving
	DoorOpen
)

func BehaviourToString(behaviour Behaviour) string {
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

func SingleElevatorFsm(
	newOrderChannel <-chan Orders,
	completedOrderChannel chan<- elevio.ButtonEvent,
	elevatorStateChannel chan<- Elevator,
	initDirection elevio.MotorDirection,
) {
	//Initializing elevator
	var state Elevator
	elevio.SetMotorDirection(initDirection)
	closestFloor := findClosestFloor()
	elevio.SetMotorDirection(elevio.MD_Stop)
	state = Elevator{Floor: closestFloor, Behaviour: Idle, Direction: elevio.MD_Stop}
	elevatorStateChannel <- state

	floorEnteredChannel := make(chan int)
	obstructedChannel := make(chan bool, 16)
	stopPressedChannel := make(chan bool, 16)
	doorTimerOutChannel := make(chan bool)
	resetTimerChannel := make(chan bool)

	go elevio.PollFloorSensor(floorEnteredChannel)
	go runTimer(configuration.DoorOpenDuration, doorTimerOutChannel, resetTimerChannel)
	go elevio.PollObstructionSwitch(obstructedChannel)
	go elevio.PollStopButton(stopPressedChannel)

	var OrderMatrix Orders

	for i := 0; i < configuration.NumFloors; i++ {
		for j := 0; j < configuration.NumButtons; j++ {
			OrderMatrix[i][j] = false
		}
	}

	for {
		select {
		case <-doorTimerOutChannel:
			switch state.Behaviour {
			case DoorOpen:
				if state.Obstructed {
					resetTimerChannel <- true
				} else {
					DirectionBehaviourPair := ordersChooseDirection(state.Floor, state.Direction, OrderMatrix)
					state.Behaviour = DirectionBehaviourPair.Behaviour
					state.Direction = Direction(DirectionBehaviourPair.Direction)
					switch state.Behaviour {
					case DoorOpen:
						resetTimerChannel <- true
						orderCompletedatCurrentFloor(state.Floor, Direction(state.Direction.convertMD()), completedOrderChannel, OrderMatrix)
					case Moving, Idle:
						elevio.SetDoorOpenLamp(false)
						elevio.SetMotorDirection(DirectionBehaviourPair.Direction)
					}
				}
			case Moving:
				panic("DoorTimerOutChannel while in Moving")
			}
		case stopbuttonpressed := <-stopPressedChannel:
			if stopbuttonpressed {
				elevio.SetStopLamp(true)
				elevio.SetMotorDirection(elevio.MD_Stop)
			} else {
				elevio.SetStopLamp(false)
			}
		case obstruction := <-obstructedChannel:
			state.Obstructed = obstruction
			if obstruction {
				if state.Behaviour == DoorOpen {
					resetTimerChannel <- true
				}
			} else {
				if state.Behaviour == DoorOpen {
					resetTimerChannel <- true
				}
			}
		case state.Floor = <-floorEnteredChannel:
			elevio.SetFloorIndicator(state.Floor)
			switch state.Behaviour {
			case Moving:
				if shouldStopAtFloor(OrderMatrix, state.Floor, state.Direction) {
					elevio.SetMotorDirection(elevio.MD_Stop)
					orderCompletedatCurrentFloor(state.Floor, Direction(state.Direction.convertMD()), completedOrderChannel, OrderMatrix)
					resetTimerChannel <- true
					state.Behaviour = DoorOpen
				}
			default:
			}
		case OrderMatrix = <-newOrderChannel:
			switch state.Behaviour {
			case Idle:
				state.Behaviour = Moving
				DirectionBehaviourPair := ordersChooseDirection(state.Floor, state.Direction, OrderMatrix)
				state.Behaviour = DirectionBehaviourPair.Behaviour
				state.Direction = Direction(DirectionBehaviourPair.Direction)
				switch state.Behaviour {
				case DoorOpen:
					resetTimerChannel <- true
					orderCompletedatCurrentFloor(state.Floor, Direction(state.Direction.convertMD()), completedOrderChannel, OrderMatrix)
				case Moving, Idle:
					elevio.SetDoorOpenLamp(false)
					elevio.SetMotorDirection(DirectionBehaviourPair.Direction)
				}
			}
		}
		elevio.SetDoorOpenLamp(state.Behaviour == DoorOpen)
		elevatorStateChannel <- state
	}
}
