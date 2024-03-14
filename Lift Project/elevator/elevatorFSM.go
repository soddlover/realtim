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
	orderAssigned <-chan Order,
	orderDelete chan<- Orderstatus,
	initElev Elev) {

	idInt, _ := strconv.Atoi(config.Self_nr)
	port := config.SimulatorPort + idInt
	addr := "localhost:" + fmt.Sprint(port)
	elevio.Init(addr, config.N_FLOORS)

	elevator := initElev

	doorTimer := time.NewTimer(config.DOOR_OPEN_TIME)
	motorErrorTimer := time.NewTimer(config.MOTOR_ERROR_TIME)
	doorTimer.Stop()
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
			case EB_UNAVAILABLE:
				if order.Button == elevio.BT_Cab && elevator.Floor == order.Floor {
					elevator.Queue[order.Floor][order.Button] = false
				}

				fmt.Println("Elevator is unavailable SO THIS NEW ORDER BETTER FUCKING BE A CAB ORDER")
			default:
				fmt.Println("Undefined state WTF")
			}

			elevatorStateBackup <- elevator
			elevatorStateBroadcast <- elevator

		case elevator.Floor = <-drv_floors:
			if elevator.State == EB_UNAVAILABLE {
				elevio.SetStopLamp(false)
				elevator.State = EB_Moving
			}

			fmt.Println("Arrived at floor", elevator.Floor)
			elevio.SetFloorIndicator(elevator.Floor)
			if ShouldStop(elevator) {
				motorErrorTimer.Stop()
				elevio.SetMotorDirection(elevio.MD_Stop)
				elevio.SetDoorOpenLamp(true)
				doorTimer.Reset(config.DOOR_OPEN_TIME)

				clearAtFloor(&elevator, orderDelete)
				//elevator.Dir = DirStop

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
				drv_floors <- elevator.Floor
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
			elevatorStateBackup <- elevator
			elevatorStateBroadcast <- elevator
			fmt.Println("Motor error, killing myself")
			elevio.SetStopLamp(true)
			for floor := 0; floor < config.N_FLOORS; floor++ {
				elevator.Queue[floor][elevio.BT_HallUp] = false
				elevator.Queue[floor][elevio.BT_HallDown] = false
			}

			//os.Exit(1)

		case obstruction := <-drv_obstr:
			if !obstruction && elevator.State == EB_UNAVAILABLE {
				doorTimer.Reset(config.DOOR_OPEN_TIME)
				elevator.Obstr = false
				elevio.SetStopLamp(false)
				elevator.State = EB_DoorOpen
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

func elevatorInit(elevator Elev, drv_floors <-chan int) Elev {
	if (elevator == Elev{}) {
		elevator = Elev{
			State: EB_Idle,
			Dir:   DirStop,
			Floor: elevio.GetFloor(),
			Queue: [config.N_FLOORS][config.N_BUTTONS]bool{},
			Obstr: false,
		}
	}
	for floor := 0; floor < config.N_FLOORS; floor++ {
		elevio.SetButtonLamp(elevio.BT_HallUp, floor, false)
		elevio.SetButtonLamp(elevio.BT_HallDown, floor, false)
	}
	if elevio.GetFloor() == -1 {
		elevio.SetMotorDirection(elevio.MD_Down)
		ticker := time.NewTicker(config.MOTOR_ERROR_TIME)
		defer ticker.Stop()
		select {
		case <-drv_floors:
			elevio.SetMotorDirection(elevio.MD_Stop)
			ticker.Stop()
			elevio.SetFloorIndicator(elevio.GetFloor())
			fmt.Println("Arrived at floor: ", elevio.GetFloor())
			elevator.State = EB_Idle
			elevator.Floor = elevio.GetFloor()
			elevio.SetMotorDirection(elevio.MotorDirection(ChooseDirection(elevator)))

		case <-ticker.C:
			fmt.Println("Failed to arrive at floor within time limit, killing myself")
			elevator.State = EB_UNAVAILABLE
			ticker.Stop()
		}
	}
	return elevator
}
