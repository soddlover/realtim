package elevator

import (
	"fmt"
	"mymodule/backup"
	. "mymodule/config"
	"mymodule/elevator/elevio"
	. "mymodule/types"
	"strconv"
	"strings"
	"time"
)

func RunElev(
	elevatorStateBroadcast chan<- Elev,
	localOrderRequest chan<- Order,
	addToQueue <-chan Order,
	orderServed chan<- Orderstatus,
	initElev Elev) {

	nr, _ := strconv.Atoi(strings.Split(SELF_ID, ":")[0])
	port := SIMULATOR_PORT + nr
	addr := "localhost:" + fmt.Sprint(port)
	elevio.Init(addr, N_FLOORS)

	elevator := initElev

	doorTimer := time.NewTimer(DOOR_OPEN_TIME)
	motorErrorTimer := time.NewTimer(MOTOR_ERROR_TIME)
	if elevator.State != EB_DoorOpen {
		doorTimer.Stop()
	}
	motorErrorTimer.Stop()

	elevatorStateBackup := make(chan Elev, ELEVATOR_BUFFER_SIZE)
	drv_buttons := make(chan elevio.ButtonEvent, ELEVATOR_BUFFER_SIZE)
	drv_floors := make(chan int, ELEVATOR_BUFFER_SIZE)
	drv_obstr := make(chan bool, ELEVATOR_BUFFER_SIZE)
	drv_stop := make(chan bool, ELEVATOR_BUFFER_SIZE)

	go backup.WriteBackup(elevatorStateBackup)
	go elevio.PollButtons(drv_buttons)
	go elevio.PollFloorSensor(drv_floors)
	go elevio.PollObstructionSwitch(drv_obstr)
	go elevio.PollStopButton(drv_stop)
	go updateLights(&elevator) //SPM: OK to use a pointer here?

	elevatorStateBackup <- elevator

	elevator = elevatorInit(elevator, drv_floors)
	if elevator.Dir != DirStop {
		motorErrorTimer.Reset(MOTOR_ERROR_TIME)
	}

	elevatorStateBackup <- elevator
	elevatorStateBroadcast <- elevator

	for {
		select {

		case buttonEvent := <-drv_buttons:
			localOrderRequest <- Order{Floor: buttonEvent.Floor, Button: buttonEvent.Button}

		case order := <-addToQueue:
			fmt.Println("Order added to local elevator queue: Floor: ", order.Floor, " Button: ", order.Button)
			elevator.Queue[order.Floor][order.Button] = true
			switch elevator.State {

			case EB_Idle:
				elevator.Dir = ChooseDirection(elevator)
				elevio.SetMotorDirection(elevio.MotorDirection(elevator.Dir))
				if elevator.Dir == DirStop {
					elevator.State = EB_DoorOpen
					doorTimer.Reset(DOOR_OPEN_TIME)
					elevio.SetDoorOpenLamp(true)
					clearAtFloor(&elevator, orderServed)

				} else {
					elevator.State = EB_Moving
					motorErrorTimer.Reset(MOTOR_ERROR_TIME)
				}

			case EB_Moving:

			case EB_DoorOpen:
				if elevator.Floor == order.Floor {
					if elevator.Queue[order.Floor][order.Button] {
						elevator.Queue[order.Floor][order.Button] = false
						orderServed <- Orderstatus{Floor: order.Floor, Button: order.Button, Served: true}
					}

					doorTimer.Reset(DOOR_OPEN_TIME)
				}

			case EB_UNAVAILABLE:
				if order.Button == BT_Cab && elevator.Floor == order.Floor && elevio.GetObstruction() {
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
				doorTimer.Reset(DOOR_OPEN_TIME)
				clearAtFloor(&elevator, orderServed)
				elevator.State = EB_DoorOpen
			} else if elevator.State == EB_Moving {
				motorErrorTimer.Reset(MOTOR_ERROR_TIME)
			}
			elevatorStateBackup <- elevator
			elevatorStateBroadcast <- elevator

		case <-doorTimer.C:
			if elevio.GetObstruction() {
				doorTimer.Reset(DOOR_OPEN_TIME)
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
			if prevdir != elevator.Dir && (elevator.Queue[elevator.Floor][BT_HallUp] || elevator.Queue[elevator.Floor][BT_HallDown]) {
				elevio.SetDoorOpenLamp(true)
				doorTimer.Reset(DOOR_OPEN_TIME)
				clearAtFloor(&elevator, orderServed)
				continue
			}
			if elevator.Dir == DirStop {
				elevator.State = EB_Idle
				motorErrorTimer.Stop()
			} else {
				elevator.State = EB_Moving
				elevio.SetMotorDirection(elevio.MotorDirection(elevator.Dir))
				motorErrorTimer.Reset(MOTOR_ERROR_TIME)
			}
			elevatorStateBackup <- elevator
			elevatorStateBroadcast <- elevator

		case <-motorErrorTimer.C:
			elevator.State = EB_UNAVAILABLE
			fmt.Println("Motor error detected")
			elevio.SetStopLamp(true)
			for floor := range elevator.Queue {
				elevator.Queue[floor][BT_HallUp] = false
				elevator.Queue[floor][BT_HallDown] = false
			}
			elevatorStateBackup <- elevator
			elevatorStateBroadcast <- elevator

		case obstruction := <-drv_obstr:
			if !obstruction && elevator.State == EB_UNAVAILABLE {
				doorTimer.Reset(DOOR_OPEN_TIME)
				elevio.SetStopLamp(false)
				elevator.State = EB_DoorOpen
				elevatorStateBackup <- elevator
				elevatorStateBroadcast <- elevator
			}

		}

	}
}
