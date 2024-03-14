package backup

import (
	"encoding/json"
	"fmt"
	"mymodule/config"
	"mymodule/elevator/elevio"
	. "mymodule/types"
	"os"
	"os/exec"
	"strings"
	"time"
)

func Backup(fresh bool) Elev {
	fmt.Println("config id:", config.Id)
	if fresh { //remove before delivery
		os.Remove("backup_" + strings.Split(config.Id, ":")[1] + ".txt") //remove before delivery
		return Elev{}                                                    //remove before delivery
	} //remove before delivery

	ticker := time.NewTicker(config.BACKUP_DEADLINE)

	for {
		<-ticker.C

		fileInfo, err := os.Stat("backup_" + strings.Split(config.Id, ":")[1] + ".txt")
		if err != nil {
			if os.IsNotExist(err) {
				// The file doesn't exist, create it
				_, err := os.Create("backup_" + strings.Split(config.Id, ":")[1] + ".txt")
				if err != nil {
					fmt.Println("Error creating file:", err)
					return takeControl()
				}
			} else {
				fmt.Println("Error getting file info:", err)
				return takeControl()
			}
		}
		currentModTime := fileInfo.ModTime()

		if time.Since(currentModTime) > config.BACKUP_DEADLINE {
			return takeControl()
		}
		fmt.Println("Process alive ...")
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
		err = os.WriteFile("backup"+strings.Split(config.Id, ":")[1]+".txt", stateJson, 0644)
		if err != nil {
			fmt.Println("Error writing to file:", err)
			continue
		}
	}
}

func takeControl() Elev {
	fmt.Println("Backup is taking over.")

	stateJson, err := os.ReadFile("backup" + strings.Split(config.Id, ":")[1] + ".txt")
	if err != nil {
		fmt.Println("Error reading file:", err)
	}
	elev := Elev{}
	err = json.Unmarshal(stateJson, &elev)
	if err != nil {
		fmt.Println("Error unmarshalling json:", err)
	} else {

		for Floor := range elev.Queue {
			elev.Queue[Floor][elevio.BT_HallUp] = false
			elev.Queue[Floor][elevio.BT_HallDown] = false
		}
	}
	cmd := exec.Command("gnome-terminal", "--", "go", "run", "main.go")
	err = cmd.Run()
	if err != nil {
		fmt.Println("Failed to spawn pair proccess, entering the DANGER ZONE")
	}
	return elev
}
