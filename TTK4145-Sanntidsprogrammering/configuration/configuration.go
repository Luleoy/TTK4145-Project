package configuration

import (
	"time"
)

const (
	NumFloors    = 4
	NumElevators = 3
	NumButtons   = 3
	Buffer       = 1024

	DisconnectTime   = 1 * time.Second
	DoorOpenDuration = 3 * time.Second
	WatchdogTime     = 5 * time.Second
	SendWVTimer      = 20 * time.Second

	PeersPort     = 16969
	BroadcastPort = 16970
)

type OrderState int

const (
	None OrderState = iota
	UnConfirmed
	//barrier everyone needs to acknowledge before going to confirmed
	Confirmed
	Completed
)

type OrderMsg struct {
	StateofOrder OrderState //state of HALL or CAB order
	AckList      map[string]bool
}
