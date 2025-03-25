package AssignerExecutable

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
)

//Assigner returns a a map with ID and hallbuttons pressed for each floor

// Struct members must be public in order to be accessible by json.Marshal/.Unmarshal
// This means they must start with a capital letter, so we need to use field renaming struct tags to make them camelCase

type HRAElevState struct {
	Behavior    string `json:"behavior"`
	Floor       int    `json:"floor"`
	Direction   string `json:"direction"`
	CabRequests []bool `json:"cabRequests"`
}

type HRAInput struct {
	HallRequests [][2]bool               `json:"hallRequests"`
	States       map[string]HRAElevState `json:"states"`
}

func PrintHRAInput(input HRAInput) {
	for k, v := range input.States {
		fmt.Println(k, " : ", v)
	}
}

func Assigner(input HRAInput) map[string][][2]bool {
	hraExecutable := ""
	switch runtime.GOOS {
	case "linux":
		hraExecutable = "hall_request_assigner"
	case "windows":
		hraExecutable = "hall_request_assigner.exe"
	default:
		panic("OS not supported")
	}

	jsonBytes, err := json.Marshal(input)
	if err != nil {
		fmt.Println("json.Marshal error: ", err)
		panic(err)
	}
	// fmt.Println("HRA INPUT:\n", string(jsonBytes))

	ret, err := exec.Command("AssignerExecutable/executables/"+hraExecutable, "-i", string(jsonBytes)).CombinedOutput()
	if err != nil {
		fmt.Println("exec.Command error: ", err)
		fmt.Println(string(ret))
		panic(err)
	}

	output := new(map[string][][2]bool)
	err = json.Unmarshal(ret, &output)
	if err != nil {
		fmt.Println("json.Unmarshal error: ", err)
		panic(err)
	}

	return *output
}

/*OUTPUT FRA HALL ASSIGNER - må sendes til order manager. Order manager må legge sammen egen matrise med matrise fra hall assigner. 1+1 skal ikke bli 2.
må sende hver av linjene til riktig heis. i order manager konverterer vi fra string til ordermatrix
{
    "0": [[false, true],
	[true, false],
	[false, false],
	[false, false]],

    "1": [[false, false],
	[false, true],
	[true, false],
	[false, true]],

    "2": [[true, false],
	[false, true],
	[false, false],
	[false, false]]
}

{
"hallRequests":
	[[false,false],[false,false],[false,false],[false,false]],
	"states":{
		"A":{"behavior":"DoorOpen","floor":1,"direction":"up","cabRequests":[false,false,false,false]},
		"C":{"behavior":"Idle","floor":0,"direction":"stop","cabRequests":[false,false,false,false]}
	}
}

*/
