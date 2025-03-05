package main

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"time"
)

const BackupAddress = "localhost:8080"
const ListenAddr = "localhost:8080"
const Timeout = 2 * time.Second

func main() {
	// ------ BACKUP MODE ------ //
	// Connect to socket, starter og lytter til master

	addr, err := net.ResolveUDPAddr("udp", ListenAddr)
	if err != nil {
		fmt.Println("Error resolving address", err)
		return
	}

	//start lytting på udp- port
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		fmt.Println("Error listening for UDP:", err)
		return
	}

	defer conn.Close() //lukk tilkobling når program avsluttet

	fmt.Println("Backup listening for heartbeat...")
	buf := make([]byte, 1024)
	count := 0

	// Listen to heartbeats in loop
	for {
		conn.SetReadDeadline(time.Now().Add(Timeout)) //hvis ikke melding før timeout, antar master død

		n, _, err := conn.ReadFromUDP(buf)
		if err != nil {
			if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
				fmt.Println("Heartbeat timed out! Taking over...")
				break // Exit loop and start counting
			}
			fmt.Println("Error receiving heartbeat:", err)
			continue
		}

		count, _ = strconv.Atoi(string(buf[:n])) // Extract count
	}

	// ------ TAKE OVER ------ //

	conn.Close() // stenger gammel tilkobling

	// Start backup in new terminal

	BackupFilePath := "~/Desktop/NewEra/TTK4145-Sanntidsprogrammering/exercise4/main.go" //MÅ ENDRE HER

	cmd := exec.Command("gnome-terminal", "--", "bash", "-c", "go run "+BackupFilePath+"; exec bash")
	error := cmd.Start()
	if error != nil {
		fmt.Println("Failed to start backup process:", err)
		return
	}

	//setter opp en ny udp_tilkobling for å sende heartbeats til neste backup
	backupAddr, err := net.ResolveUDPAddr("udp", BackupAddress)
	if err != nil {
		fmt.Println("Failed to resolve backup address:", err)
		os.Exit(1)
	}

	conn, err = net.DialUDP("udp", nil, backupAddr) // oppdretter en ny forbindelse for å sende ut heartbeats
	if err != nil {
		fmt.Println("Failed to connect to backup process:", err)
		os.Exit(1)
	}
	defer conn.Close()

	// ------- PRIMARY MODE ------- //ny master og sender ut heartbeats

	count++
	for {
		// Send a heartbeat with count
		msg := fmt.Sprintf("%d", count)
		_, err := conn.Write([]byte(msg))
		if err != nil {
			fmt.Println("Failed to send heartbeat:", err)
		}
		// Print the count
		fmt.Println(count)
		time.Sleep(time.Second)
		count++
	}
}
