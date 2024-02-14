package main

import (
	"fmt"
	. "mymodule/elevator"
)

func main() {
	channels := Channels{
		ElevatorStates: make(chan Elev),
		OrderRequest:   make(chan Order),
		OrderComplete:  make(chan Order),
		OrderAssigned:  make(chan Order),
	}
	fmt.Print("Hello, World!")
	go RunElev(channels)

	select {}
}
