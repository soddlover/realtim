package main

import (
	"fmt"
	elevatorFSM "mymodule/elevator"
)

func main() {
	fmt.Println("Hello, World!")
	go elevatorFSM.RunElev()
	select {}
}
