package elevatorFSM

import (
	"fmt"
	. "mymodule/config"
	"mymodule/elevator/elevio"
	"time"
)

type ElevatorState int

type Channels struct {
	ElevatorStates chan Elev
	OrderRequest   chan Order
	OrderComplete  chan Order
	OrderAssigned  chan Order
}

const (
	EB_Idle ElevatorState = iota
	EB_Moving
	EB_DoorOpen
	Undefined
)

type ElevatorDirection int

const (
	DirUp   ElevatorDirection = 1
	DirDown                   = -1
	DirStop                   = 0
)

type Elev struct {
	State ElevatorState
	Dir   ElevatorDirection
	Floor int
	Queue [N_FLOORS][N_BUTTONS]bool
}

type Order struct {
	Floor  int
	Button elevio.ButtonType
}

func RunElev(channels Channels) {
	elevio.Init("localhost:15657", N_FLOORS)

	elevator := Elev{
		State: EB_Idle,
		Dir:   DirStop,
		Floor: elevio.GetFloor(),
		Queue: [N_FLOORS][N_BUTTONS]bool{},
	}
	doorTimer := time.NewTimer(3 * time.Second)
	doorTimer.Stop()
	motorErrorTimer := time.NewTimer(3 * time.Second)
	motorErrorTimer.Stop()

	drv_buttons := make(chan elevio.ButtonEvent, 10)
	drv_floors := make(chan int)
	drv_obstr := make(chan bool)
	drv_stop := make(chan bool)

	go elevio.PollButtons(drv_buttons)
	go elevio.PollFloorSensor(drv_floors)
	go elevio.PollObstructionSwitch(drv_obstr)
	go elevio.PollStopButton(drv_stop)

	elevStart(drv_floors, elevator)

	for {
		select {
		case buttonEvent := <-drv_buttons:
			fmt.Println("Button event at floor", buttonEvent.Floor, "button", buttonEvent.Button)

			channels.OrderRequest <- Order{buttonEvent.Floor, buttonEvent.Button}

			//elevator.Queue[buttonEvent.Floor][buttonEvent.Button] = true
			//elevio.SetButtonLamp(buttonEvent.Button, buttonEvent.Floor, true)

		case order := <-channels.OrderAssigned:
			fmt.Println("Order assigned: ", order)
			elevator.Queue[order.Floor][order.Button] = true
			elevio.SetButtonLamp(order.Button, order.Floor, true)
			switch elevator.State {
			case EB_Idle:
				elevator.Dir = ChooseDirection(elevator)
				elevio.SetMotorDirection(elevio.MotorDirection(elevator.Dir))
				if elevator.Dir == DirStop {
					elevator.State = EB_DoorOpen
					doorTimer.Reset(3 * time.Second)
					elevio.SetDoorOpenLamp(true)
					elevator.Queue[elevator.Floor] = [N_BUTTONS]bool{}
					elevio.SetButtonLamp(elevio.BT_Cab, elevator.Floor, false)
				} else {
					elevator.State = EB_Moving
					motorErrorTimer.Reset(3 * time.Second)
				}
			case EB_Moving:
			case EB_DoorOpen:
				if elevator.Floor == order.Floor {
					elevator.Queue[order.Floor][order.Button] = false
					elevio.SetButtonLamp(order.Button, order.Floor, false)
					doorTimer.Reset(3 * time.Second)
				}
			case Undefined:
			default:
				fmt.Println("Undefined state WTF")
			}

			channels.ElevatorStates <- elevator

		/*
			case order := <-channels.orderComplete:
				elevio.SetButtonLamp(order.Button, order.Floor, false)
				//add func for only one way clearing later
		*/
		case elevator.Floor = <-drv_floors:
			fmt.Println("Arrived at floor", elevator.Floor)
			elevio.SetFloorIndicator(elevator.Floor)
			if ShouldStop(elevator) {
				motorErrorTimer.Stop()
				elevio.SetMotorDirection(elevio.MD_Stop)
				elevio.SetDoorOpenLamp(true)
				doorTimer.Reset(3 * time.Second)
				elevator.Queue[elevator.Floor] = [N_BUTTONS]bool{}
				elevio.SetButtonLamp(elevio.BT_Cab, elevator.Floor, false)
				elevio.SetButtonLamp(elevio.BT_HallUp, elevator.Floor, false)
				elevio.SetButtonLamp(elevio.BT_HallDown, elevator.Floor, false)
				//same here add one way clearing later
				//channels.orderComplete <- Order{elevator.Floor, elevio.BT_HallUp}
				//channels.orderComplete <- Order{elevator.Floor, elevio.BT_HallDown}
				elevator.State = EB_DoorOpen
			} else if elevator.State == EB_Moving {
				motorErrorTimer.Reset(3 * time.Second)
			}
			channels.ElevatorStates <- elevator

		case <-doorTimer.C:
			elevio.SetDoorOpenLamp(false)
			elevator.Dir = ChooseDirection(elevator)
			if elevator.Dir == DirStop {
				elevator.State = EB_Idle
				motorErrorTimer.Stop()
			} else {
				elevator.State = EB_Moving
				elevio.SetMotorDirection(elevio.MotorDirection(elevator.Dir))
				motorErrorTimer.Reset(3 * time.Second)
			}
			channels.ElevatorStates <- elevator
		case <-motorErrorTimer.C:
			elevio.SetMotorDirection(elevio.MD_Stop)
			elevator.State = Undefined
			fmt.Println("Motor error")
			for i := 0; i < 10; i++ {
				elevio.SetStopLamp(true)
				time.Sleep(500 * time.Millisecond)
				elevio.SetStopLamp(false)
				time.Sleep(500 * time.Millisecond)
			}
			elevStart(drv_floors, elevator)
			elevator.State = EB_Idle
			channels.ElevatorStates <- elevator
			//fullfør cab ordre
		}

	}
}
