package singleElevator

import (
	"TTK4145-Heislab/driver-go/elevio"
)

type Direction int

const (
	Down Direction = -1
	Up   Direction = 1
	Stop Direction = 0
)

func (d Direction) convertMD() elevio.MotorDirection {
	return map[Direction]elevio.MotorDirection{Up: elevio.MD_Up, Down: elevio.MD_Down}[d]
}
