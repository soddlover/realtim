package elevatorFSM

import (
	"fmt"
	"mymodule/config"
	"mymodule/elevator/elevio"
	. "mymodule/types"
	"strconv"
	"time"
)

func RunElev(
	elevatorStateBackup chan<- Elev,
	elevatorStateBroadcast chan<- Elev,
	orderRequest chan<- Order,
	orderAssigned <-chan Orderstatus,
	orderDelete chan<- Orderstatus,
	initElev Elev) {

	idInt, _ := strconv.Atoi(config.Self_nr)
	port := config.SimulatorPort + idInt
	addr := "localhost:" + fmt.Sprint(port)
	elevio.Init(addr, config.N_FLOORS)

	elevator := initElev
	if (elevator == Elev{}) {
		elevator = Elev{
			State: EB_Idle,
			Dir:   DirStop,
			Floor: elevio.GetFloor(),
			Queue: [config.N_FLOORS][config.N_BUTTONS]bool{},
			Obstr: false,
		}
	}

	doorTimer := time.NewTimer(config.DOOR_OPEN_TIME)
	doorTimer.Stop()
	motorErrorTimer := time.NewTimer(config.MOTOR_ERROR_TIME)
	motorErrorTimer.Stop()

	drv_buttons := make(chan elevio.ButtonEvent, 10)
	drv_floors := make(chan int, 10)
	drv_obstr := make(chan bool, 10)
	drv_stop := make(chan bool, 10)
	go elevio.PollButtons(drv_buttons)
	go elevio.PollFloorSensor(drv_floors)
	go elevio.PollObstructionSwitch(drv_obstr)
	go elevio.PollStopButton(drv_stop)
	go updateLights(&elevator)
	elevStart(drv_floors)
	elevator.Floor = elevio.GetFloor()
	elevio.SetMotorDirection(elevio.MotorDirection(ChooseDirection(elevator)))
	elevatorStateBackup <- elevator
	elevatorStateBroadcast <- elevator

	for {
		select {
		case buttonEvent := <-drv_buttons:
			fmt.Println("Button event at floor", buttonEvent.Floor, "button", buttonEvent.Button)
			orderRequest <- Order{Floor: buttonEvent.Floor, Button: buttonEvent.Button}

		case order := <-orderAssigned:
			fmt.Println("Order assigned: ", order)
			elevator.Queue[order.Floor][order.Button] = true
			switch elevator.State {
			case EB_Idle:
				elevator.Dir = ChooseDirection(elevator)
				elevio.SetMotorDirection(elevio.MotorDirection(elevator.Dir))
				if elevator.Dir == DirStop {
					elevator.State = EB_DoorOpen
					doorTimer.Reset(config.DOOR_OPEN_TIME)
					elevio.SetDoorOpenLamp(true)
					clearAtFloor(&elevator, orderDelete)

				} else {
					elevator.State = EB_Moving
					motorErrorTimer.Reset(config.MOTOR_ERROR_TIME)
				}
			case EB_Moving:
			case EB_DoorOpen:
				if elevator.Floor == order.Floor {
					if elevator.Queue[order.Floor][order.Button] {
						elevator.Queue[order.Floor][order.Button] = false
						orderDelete <- Orderstatus{Owner: config.Self_nr, Floor: order.Floor, Button: order.Button, Served: true}
					}

					doorTimer.Reset(config.DOOR_OPEN_TIME)
				}
			case Undefined:
			default:
				fmt.Println("Undefined state WTF")
			}

			elevatorStateBackup <- elevator
			elevatorStateBroadcast <- elevator

		case elevator.Floor = <-drv_floors:
			fmt.Println("Arrived at floor", elevator.Floor)
			elevio.SetFloorIndicator(elevator.Floor)
			if ShouldStop(elevator) {
				motorErrorTimer.Stop()
				elevio.SetMotorDirection(elevio.MD_Stop)
				elevio.SetDoorOpenLamp(true)
				doorTimer.Reset(config.DOOR_OPEN_TIME)

				clearAtFloor(&elevator, orderDelete)
				elevator.Dir = DirStop

				elevator.State = EB_DoorOpen
			} else if elevator.State == EB_Moving {
				motorErrorTimer.Reset(config.MOTOR_ERROR_TIME)
			}
			elevatorStateBackup <- elevator
			elevatorStateBroadcast <- elevator

		case <-doorTimer.C:
			if elevio.GetObstruction() {
				doorTimer.Reset(config.DOOR_OPEN_TIME)
				elevator.Obstr = true
				fmt.Println("Obstruction detected")
				elevatorStateBackup <- elevator
				elevatorStateBroadcast <- elevator
				continue
			}
			elevio.SetDoorOpenLamp(false)
			elevator.Dir = ChooseDirection(elevator)
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
			elevio.SetMotorDirection(elevio.MD_Stop)
			elevator.State = Undefined
			fmt.Println("Motor error")
			for i := 0; i < 10; i++ {
				elevio.SetStopLamp(true)
				time.Sleep(500 * time.Millisecond)
				elevio.SetStopLamp(false)
				time.Sleep(500 * time.Millisecond)
			}
			elevStart(drv_floors)
			elevator.State = EB_Idle
			elevatorStateBackup <- elevator
			elevatorStateBroadcast <- elevator
		//fullfør cab ordre
		case obstruction := <-drv_obstr:
			if !obstruction && elevator.Obstr {
				doorTimer.Reset(config.DOOR_OPEN_TIME)
				elevator.Obstr = false
				elevatorStateBackup <- elevator
				elevatorStateBroadcast <- elevator
			}

		}

	}
}
func updateLights(elevator *Elev) {
	for {
		for floor := 0; floor < config.N_FLOORS; floor++ {
			for button := 0; button < config.N_BUTTONS; button++ {
				if elevio.ButtonType(button) == elevio.BT_Cab {
					elevio.SetButtonLamp(elevio.ButtonType(button), floor, elevator.Queue[floor][button])
				}
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
}
func UpdateLightsFromNetworkOrders(networkorders [config.N_FLOORS][config.N_BUTTONS]string) {
	for floor := 0; floor < config.N_FLOORS; floor++ {
		for button := 0; button < config.N_BUTTONS; button++ {
			if elevio.ButtonType(button) != elevio.BT_Cab {
				if networkorders[floor][button] != "" {
					elevio.SetButtonLamp(elevio.ButtonType(button), floor, true)
				} else {
					elevio.SetButtonLamp(elevio.ButtonType(button), floor, false)
				}
			}
		}
	}
}
