package elevator

import (
	"fmt"
	"mymodule/config"
	"mymodule/elevator/elevio"
	. "mymodule/types"
	"strconv"
	"strings"
	"time"
)

func RunElev(
	elevatorStateBackup chan<- Elev,
	elevatorStateBroadcast chan<- Elev,
	localOrderRequest chan<- Order,
	addToQueue <-chan Order,
	orderServed chan<- Orderstatus,
	initElev Elev) {

	nr, _ := strconv.Atoi(strings.Split(config.Id, ":")[0]) //remove before delivery
	port := config.SimulatorPort + nr
	addr := "localhost:" + fmt.Sprint(port)
	elevio.Init(addr, N_FLOORS)

	elevator := initElev

	doorTimer := time.NewTimer(config.DOOR_OPEN_TIME)
	motorErrorTimer := time.NewTimer(config.MOTOR_ERROR_TIME)
	doorTimer.Stop()
	motorErrorTimer.Stop()

	drv_buttons := make(chan elevio.ButtonEvent, config.ELEVATOR_BUFFER_SIZE)
	drv_floors := make(chan int, config.ELEVATOR_BUFFER_SIZE)
	drv_obstr := make(chan bool, config.ELEVATOR_BUFFER_SIZE)
	drv_stop := make(chan bool, config.ELEVATOR_BUFFER_SIZE)

	go elevio.PollButtons(drv_buttons)
	go elevio.PollFloorSensor(drv_floors)
	go elevio.PollObstructionSwitch(drv_obstr)
	go elevio.PollStopButton(drv_stop)
	go updateLights(&elevator) //SPM: OK to use a pointer here?

	elevatorStateBackup <- elevator

	elevator = elevatorInit(elevator, drv_floors)
	if elevator.Dir != DirStop {
		motorErrorTimer.Reset(config.MOTOR_ERROR_TIME)
	}

	elevatorStateBackup <- elevator
	elevatorStateBroadcast <- elevator

	for {
		select {
		case buttonEvent := <-drv_buttons:
			fmt.Println("buttonevent on", buttonEvent.Floor)
			localOrderRequest <- Order{Floor: buttonEvent.Floor, Button: buttonEvent.Button}

		case order := <-addToQueue:
			elevator.Queue[order.Floor][order.Button] = true
			switch elevator.State {

			case EB_Idle:
				elevator.Dir = ChooseDirection(elevator)
				elevio.SetMotorDirection(elevio.MotorDirection(elevator.Dir))
				if elevator.Dir == DirStop {
					elevator.State = EB_DoorOpen
					doorTimer.Reset(config.DOOR_OPEN_TIME)
					elevio.SetDoorOpenLamp(true)
					clearAtFloor(&elevator, orderServed)

				} else {
					elevator.State = EB_Moving
					motorErrorTimer.Reset(config.MOTOR_ERROR_TIME)
				}

			case EB_Moving:

			case EB_DoorOpen:
				if elevator.Floor == order.Floor {
					if elevator.Queue[order.Floor][order.Button] {
						elevator.Queue[order.Floor][order.Button] = false
						orderServed <- Orderstatus{Floor: order.Floor, Button: order.Button, Served: true}
					}

					doorTimer.Reset(config.DOOR_OPEN_TIME)
				}

			case EB_UNAVAILABLE:
				if order.Button == elevio.BT_Cab && elevator.Floor == order.Floor && elevio.GetObstruction() {
					elevator.Queue[order.Floor][order.Button] = false
				}
			}

			elevatorStateBackup <- elevator
			elevatorStateBroadcast <- elevator

		case elevator.Floor = <-drv_floors:
			if elevator.State == EB_UNAVAILABLE {
				elevio.SetStopLamp(false)
				elevator.State = EB_Moving
			}
			elevio.SetFloorIndicator(elevator.Floor)
			if ShouldStop(elevator) {
				motorErrorTimer.Stop()
				elevio.SetMotorDirection(elevio.MD_Stop)
				elevio.SetDoorOpenLamp(true)
				doorTimer.Reset(config.DOOR_OPEN_TIME)
				clearAtFloor(&elevator, orderServed)
				elevator.State = EB_DoorOpen
			} else if elevator.State == EB_Moving {
				motorErrorTimer.Reset(config.MOTOR_ERROR_TIME)
			}
			elevatorStateBackup <- elevator
			elevatorStateBroadcast <- elevator
		case <-doorTimer.C:
			if elevio.GetObstruction() {
				doorTimer.Reset(config.DOOR_OPEN_TIME)
				elevio.SetStopLamp(true)
				elevator.State = EB_UNAVAILABLE
				fmt.Println("Obstruction detected")
				elevatorStateBackup <- elevator
				elevatorStateBroadcast <- elevator
				continue
			}
			elevio.SetDoorOpenLamp(false)
			prevdir := elevator.Dir
			elevator.Dir = ChooseDirection(elevator)
			if prevdir != elevator.Dir && (elevator.Queue[elevator.Floor][elevio.BT_HallUp] || elevator.Queue[elevator.Floor][elevio.BT_HallDown]) {
				elevio.SetDoorOpenLamp(true)
				doorTimer.Reset(config.DOOR_OPEN_TIME)
				clearAtFloor(&elevator, orderServed)
				fmt.Println("BOomb booomm baby changing direction")
				continue
			}
			if elevator.Dir == DirStop {
				elevator.State = EB_Idle
				motorErrorTimer.Stop()
			} else {
				elevator.State = EB_Moving
				elevio.SetMotorDirection(elevio.MotorDirection(elevator.Dir))
				motorErrorTimer.Reset(config.MOTOR_ERROR_TIME)
			}
			elevatorStateBackup <- elevator
			elevatorStateBroadcast <- elevator
		case <-motorErrorTimer.C:
			elevator.State = EB_UNAVAILABLE
			fmt.Println("Motor error, killing myself")
			elevio.SetStopLamp(true)
			for floor := range elevator.Queue {
				elevator.Queue[floor][elevio.BT_HallUp] = false
				elevator.Queue[floor][elevio.BT_HallDown] = false
			}
			elevatorStateBackup <- elevator
			elevatorStateBroadcast <- elevator

		case obstruction := <-drv_obstr:
			if !obstruction && elevator.State == EB_UNAVAILABLE {
				doorTimer.Reset(config.DOOR_OPEN_TIME)
				elevio.SetStopLamp(false)
				elevator.State = EB_DoorOpen
				elevatorStateBackup <- elevator
				elevatorStateBroadcast <- elevator
			}

		}

	}
}
