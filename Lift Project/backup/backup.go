package backup

import (
	"encoding/json"
	"fmt"
	"mymodule/config"
	. "mymodule/config"
	"mymodule/elevator/elevio"
	. "mymodule/types"
	"os"
	"time"
)

func Backup(fresh bool) Elev {
	// Get the initial file info

	if fresh {
		os.Remove("backup" + config.Self_nr + ".txt")
		return Elev{}
	}
	// Start a ticker that checks the file every second
	ticker := time.NewTicker(10 * time.Second)

	for {
		<-ticker.C

		// Get the current file info
		fileInfo, err := os.Stat("backup" + config.Self_nr + ".txt")
		if err != nil {
			fmt.Println("Error getting file info:", err)
			return takeControl()

		}

		// Get the current modification time
		currentModTime := fileInfo.ModTime()

		// If the file has not been modified in the last 10 seconds, the backup takes over
		if time.Since(currentModTime) > 10*time.Second {
			return takeControl()
		}
		fmt.Println("Backup is still alive. KJØH ")

	}
}

func WriteBackup(elevChan <-chan Elev) {
	ticker := time.NewTicker(5 * time.Second)
	var elev Elev

	for {
		select {
		case elev = <-elevChan:
		case <-ticker.C:
		}
		stateJson, err := json.Marshal(elev)
		if err != nil {
			fmt.Println("Error marshalling json:", err)
			continue
		}
		err = os.WriteFile("backup"+config.Self_nr+".txt", stateJson, 0644)
		if err != nil {
			fmt.Println("Error writing to file:", err)
			continue

		}
	}
}

func takeControl() Elev {
	fmt.Println("Backup is taking over.")

	stateJson, err := os.ReadFile("backup" + config.Self_nr + ".txt")
	if err != nil {
		fmt.Println("Error reading file:", err)
	}
	elev := Elev{}
	err = json.Unmarshal(stateJson, &elev)
	if err != nil {
		fmt.Println("Error unmarshalling json:", err)
	} else {
		for Floor := 0; Floor < N_FLOORS; Floor++ {
			if Floor < len(elev.Queue) && int(elevio.BT_HallUp) < len(elev.Queue[Floor]) && int(elevio.BT_HallDown) < len(elev.Queue[Floor]) {
				elev.Queue[Floor][elevio.BT_HallUp] = false
				elev.Queue[Floor][elevio.BT_HallDown] = false
			} else {
				fmt.Println("Index out of range")
			}
		}
	}

	// cmd := exec.Command("gnome-terminal", "--", "go", "run", "main.go")
	// err = cmd.Run()
	// if err != nil {
	// 	fmt.Println("THis sucks")
	// 	log.Fatal(err)
	// }
	// Here you can add the code for the backup to take over
	return elev
}
