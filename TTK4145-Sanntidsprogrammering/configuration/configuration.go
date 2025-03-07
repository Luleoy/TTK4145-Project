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
)

type OrderState int

const (
	None OrderState = iota
	UnConfirmed
	// barrier everyone needs acknowlade before going to confirmed
	Confirmed
	Complete
)

type OrderMsg struct {
	StateofOrder OrderState //state of HALL or CAB order
	AckList      map[string]bool
}

//hva skal vi gjøre med numPeers?

//legge typen i configuration. Lage kanalene de skal sendes på i main.g. structuren på hva som blir sendt på kanalen

//hva skla detligge i const, struct?
