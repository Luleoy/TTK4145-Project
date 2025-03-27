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
	SendWVTimer      = 10 * time.Millisecond

	PeersPort     = 16258
	BroadcastPort = 16091
)

type OrderState int

const (
	None OrderState = iota
	UnConfirmed
	Confirmed
	Completed
)

type OrderMessage struct {
	StateofOrder OrderState
	AckList      map[string]bool
}
