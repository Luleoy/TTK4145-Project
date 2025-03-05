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

type RequestState int

const (
	None RequestState = iota
	Order
	Confirmed
	Complete
)

//hva skal vi gjøre med numPeers?

//legge typen i configuration. Lage kanalene de skal sendes på i main.g. structuren på hva som blir sendt på kanalen

//hva skla detligge i const, struct?
