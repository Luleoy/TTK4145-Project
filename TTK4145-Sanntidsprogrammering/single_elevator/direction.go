package single_elevator

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

func (d Direction) convertBT() elevio.ButtonType {
	return map[Direction]elevio.ButtonType{Up: elevio.BT_HallUp, Down: elevio.BT_HallDown}[d]
}

// invert motordirection to get opposite direction
func (d Direction) invertMD() Direction {
	return map[Direction]Direction{Up: Down, Down: Up}[d]
}

func (d Direction) MotorDirectionToString() string {
	return map[Direction]string{Up: "up", Down: "down"}[d]
}
