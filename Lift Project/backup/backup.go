package backup

import (
	"encoding/json"
	"fmt"
	"log"
	. "mymodule/config"
	. "mymodule/types"
	"os"
	"os/exec"
	"strings"
	"time"
)

func Backup(fresh bool) Elev {

	if fresh {
		os.Remove("backup" + strings.Split(SELF_ID, ":")[1] + ".txt")
		return Elev{}
	}

	ticker := time.NewTicker(BACKUP_DEADLINE)

	for {
		<-ticker.C

		fileInfo, err := os.Stat("backup" + strings.Split(SELF_ID, ":")[1] + ".txt")
		if err != nil {
			fmt.Println("Error getting file info:", err)
			return takeControl()
		}

		currentModTime := fileInfo.ModTime()

		if time.Since(currentModTime) > BACKUP_DEADLINE {
			return takeControl()
		}
		fmt.Println("Process is still alive, standing by...")
	}
}

func WriteBackup(elevChan <-chan Elev) {

	ticker := time.NewTicker(BACKUP_INTERVAL)
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
		err = os.WriteFile("backup"+strings.Split(SELF_ID, ":")[1]+".txt", stateJson, 0644)
		if err != nil {
			fmt.Println("Error writing to file:", err)
			continue
		}
	}
}

func takeControl() Elev {

	fmt.Println("Backup is taking over.")

	stateJson, err := os.ReadFile("backup" + strings.Split(SELF_ID, ":")[1] + ".txt")
	if err != nil {
		fmt.Println("Error reading file:", err)
	}
	elev := Elev{}
	err = json.Unmarshal(stateJson, &elev)
	if err != nil {
		fmt.Println("Error unmarshalling json:", err)
	} else {
		for Floor := range elev.Queue {
			elev.Queue[Floor][BT_HallUp] = false
			elev.Queue[Floor][BT_HallDown] = false
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
