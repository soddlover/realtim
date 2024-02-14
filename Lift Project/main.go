package main

import (
	"fmt"
	. "mymodule/assigner"
	. "mymodule/elevator"
)

func main() {
	world := &World{
		Map: make(map[string]Elev),
	}

	channels := Channels{
		ElevatorStates: make(chan Elev),
		OrderRequest:   make(chan Order),
		OrderComplete:  make(chan Order),
		OrderAssigned:  make(chan Order),
	}
	fmt.Print("Hello, World!")
	go RunElev(channels)
	go Assigner(channels, world)
	go print(channels)
	select {}
}

func print(channels Channels) {
	for {
		select {

		case elev := <-channels.ElevatorStates:
			fmt.Println("Elevator state: ", elev)
		}
	}
}
