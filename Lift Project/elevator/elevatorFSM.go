package elevatorFSM

import (
	"fmt"
	. "mymodule/config"
	"mymodule/elevator/elevio"
	"time"
)

type ElevatorState int

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

func RunElev() {
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

	drv_buttons := make(chan elevio.ButtonEvent)
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
			fmt.Println("Button event at floor", buttonEvent.Floor+1, "button", buttonEvent.Button)
			elevator.Queue[buttonEvent.Floor][buttonEvent.Button] = true
			elevio.SetButtonLamp(buttonEvent.Button, buttonEvent.Floor, true)

			switch elevator.State {
			case EB_Idle:
				elevator.Dir = chooseDirection(elevator)
				elevio.SetMotorDirection(elevio.MotorDirection(elevator.Dir))
				if elevator.Dir == DirStop {
					elevator.State = EB_DoorOpen
					doorTimer.Reset(3 * time.Second)
					elevio.SetDoorOpenLamp(true)
					elevator.Queue[elevator.Floor] = [N_BUTTONS]bool{}
				} else {
					elevator.State = EB_Moving
					motorErrorTimer.Reset(3 * time.Second)
				}

			case EB_Moving:
			case EB_DoorOpen:
				if elevator.Floor == buttonEvent.Floor {
					elevator.Queue[buttonEvent.Floor][buttonEvent.Button] = false
					elevio.SetButtonLamp(buttonEvent.Button, buttonEvent.Floor, false)
					doorTimer.Reset(3 * time.Second)
				}
			case Undefined:
			default:
				fmt.Println("Undefined state WTF")
			}

		case elevator.Floor = <-drv_floors:
			fmt.Println("Arrived at floor", elevator.Floor+1)
			elevio.SetFloorIndicator(elevator.Floor)
			if shouldStop(elevator) {
				motorErrorTimer.Stop()
				elevio.SetMotorDirection(elevio.MD_Stop)
				elevio.SetDoorOpenLamp(true)
				doorTimer.Reset(3 * time.Second)
				elevator.Queue[elevator.Floor] = [N_BUTTONS]bool{}
				elevio.SetButtonLamp(elevio.BT_HallUp, elevator.Floor, false)
				elevio.SetButtonLamp(elevio.BT_HallDown, elevator.Floor, false)
				elevio.SetButtonLamp(elevio.BT_Cab, elevator.Floor, false)
				elevator.State = EB_DoorOpen
			} else if elevator.State == EB_Moving {
				motorErrorTimer.Reset(3 * time.Second)
			}

		case <-doorTimer.C:
			elevio.SetDoorOpenLamp(false)
			elevator.Dir = chooseDirection(elevator)
			if elevator.Dir == DirStop {
				elevator.State = EB_Idle
				motorErrorTimer.Stop()
			} else {
				elevator.State = EB_Moving
				elevio.SetMotorDirection(elevio.MotorDirection(elevator.Dir))
				motorErrorTimer.Reset(3 * time.Second)
			}
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
			//fullfør cab ordre

		}
	}
}
