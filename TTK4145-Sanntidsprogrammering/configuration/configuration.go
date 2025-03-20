package configuration

import (
	"time"
)

const (
	NumFloors  = 4
	NumButtons = 3
	Buffer     = 100

	DisconnectTime   = 1 * time.Second
	DoorOpenDuration = 3 * time.Second
	WatchdogTime     = 5 * time.Second
	SendWVTimer      = 1000 * time.Millisecond //for at lyset skal være raskere, gjør denne mindre, er bare stor for debugging

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
