package backup

import (
	"encoding/json"
	"fmt"
	"log"
	"mymodule/config"
	. "mymodule/config"
	"mymodule/elevator/elevio"
	. "mymodule/types"
	"os"
	"os/exec"
	"time"
)

func Backup(fresh bool) Elev {

	if fresh {
		os.Remove("backup" + "0" + ".txt")
		return Elev{}
	}

	ticker := time.NewTicker(config.BACKUP_DEADLINE)

	for {
		<-ticker.C

		fileInfo, err := os.Stat("backup" + "0" + ".txt")
		if err != nil {
			fmt.Println("Error getting file info:", err)
			return takeControl()
		}

		currentModTime := fileInfo.ModTime()

		if time.Since(currentModTime) > config.BACKUP_DEADLINE {
			return takeControl()
		}
		fmt.Println("Backup is still alive. KJØH ")
	}
}

func WriteBackup(elevChan <-chan Elev) {

	ticker := time.NewTicker(config.BACKUP_INTERVAL)
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
		err = os.WriteFile("backup"+"0"+".txt", stateJson, 0644)
		if err != nil {
			fmt.Println("Error writing to file:", err)
			continue
		}
	}
}

func takeControl() Elev {
	fmt.Println("Backup is taking over.")

	stateJson, err := os.ReadFile("backup" + "0" + ".txt")
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
	cmd := exec.Command("gnome-terminal", "--", "go", "run", "main.go")
	err = cmd.Run()
	if err != nil {
		fmt.Println("THis sucks")
		log.Fatal(err)
	}
	return elev
}
