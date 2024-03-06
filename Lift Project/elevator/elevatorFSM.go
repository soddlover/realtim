package elevatorFSM

import (
	"fmt"
	"mymodule/config"
	. "mymodule/config"
	"mymodule/elevator/elevio"
	. "mymodule/types"
	"strconv"
	"time"
)

func RunElev(channels Channels, initElev Elev) {
	idInt, _ := strconv.Atoi(config.Self_nr)
	port := 15657 + idInt
	addr := "localhost:" + fmt.Sprint(port)
	elevio.Init(addr, N_FLOORS)

	elevator := initElev
	if (elevator == Elev{}) {
		elevator = Elev{
			State: EB_Idle,
			Dir:   DirStop,
			Floor: elevio.GetFloor(),
			Queue: [N_FLOORS][N_BUTTONS]bool{},
			Obstr: false,
		}
	}

	doorTimer := time.NewTimer(3 * time.Second)
	doorTimer.Stop()
	motorErrorTimer := time.NewTimer(3 * time.Second)
	motorErrorTimer.Stop()

	drv_buttons := make(chan elevio.ButtonEvent, 10)
	drv_floors := make(chan int, 10)
	drv_obstr := make(chan bool, 10)
	drv_stop := make(chan bool, 10)
	ordersToQueue := make(chan Order, 10)
	go saveLocalOrder(channels, ordersToQueue)

	go elevio.PollButtons(drv_buttons)
	go elevio.PollFloorSensor(drv_floors)
	go elevio.PollObstructionSwitch(drv_obstr)
	go elevio.PollStopButton(drv_stop)
	go updateLights(&elevator)
	//go printElevator(&elevator)
	elevStart(drv_floors)
	elevio.SetMotorDirection(elevio.MotorDirection(ChooseDirection(elevator)))

	for {
		select {
		case buttonEvent := <-drv_buttons:
			fmt.Println("Button event at floor", buttonEvent.Floor, "button", buttonEvent.Button)

			channels.OrderRequest <- Order{buttonEvent.Floor, buttonEvent.Button}

			//elevator.Queue[buttonEvent.Floor][buttonEvent.Button] = true
			//elevio.SetButtonLamp(buttonEvent.Button, buttonEvent.Floor, true)

		case order := <-ordersToQueue:
			fmt.Println("Order assigned: ", order)
			elevator.Queue[order.Floor][order.Button] = true
			switch elevator.State {
			case EB_Idle:
				elevator.Dir = ChooseDirection(elevator)
				elevio.SetMotorDirection(elevio.MotorDirection(elevator.Dir))
				if elevator.Dir == DirStop {
					elevator.State = EB_DoorOpen
					doorTimer.Reset(3 * time.Second)
					elevio.SetDoorOpenLamp(true)
					elevator.Queue[elevator.Floor] = [N_BUTTONS]bool{}
					channels.OrderComplete <- Order{elevator.Floor, elevio.BT_HallUp}
					channels.OrderComplete <- Order{elevator.Floor, elevio.BT_HallDown}
				} else {
					elevator.State = EB_Moving
					motorErrorTimer.Reset(3 * time.Second)
				}
			case EB_Moving:
			case EB_DoorOpen:
				if elevator.Floor == order.Floor {
					elevator.Queue[order.Floor][order.Button] = false
					channels.OrderComplete <- order
					doorTimer.Reset(3 * time.Second)
				}
			case Undefined:
			default:
				fmt.Println("Undefined state WTF")
			}

			channels.ElevatorStates <- elevator
			channels.ElevatorStatesBroadcast <- elevator

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
				//clear only orders in correct direction
				//maybe put this in a function
				if elevator.Dir == DirUp {
					elevator.Queue[elevator.Floor][elevio.BT_HallUp] = false
					channels.OrderComplete <- Order{elevator.Floor, elevio.BT_HallUp}
					if !OrdersAbove(elevator) {
						elevator.Queue[elevator.Floor][elevio.BT_HallDown] = false
						channels.OrderComplete <- Order{elevator.Floor, elevio.BT_HallDown}
					}
				} else if elevator.Dir == DirDown {
					elevator.Queue[elevator.Floor][elevio.BT_HallDown] = false
					channels.OrderComplete <- Order{elevator.Floor, elevio.BT_HallDown}
					if !OrdersBelow(elevator) {
						elevator.Queue[elevator.Floor][elevio.BT_HallUp] = false
						channels.OrderComplete <- Order{elevator.Floor, elevio.BT_HallUp}
					}
				}
				elevator.Dir = DirStop

				elevator.Queue[elevator.Floor][elevio.BT_Cab] = false
				//same here add one way clearing later
				//channels.orderComplete <- Order{elevator.Floor, elevio.BT_HallUp}
				//channels.orderComplete <- Order{elevator.Floor, elevio.BT_HallDown}
				elevator.State = EB_DoorOpen
			} else if elevator.State == EB_Moving {
				motorErrorTimer.Reset(3 * time.Second)
			}
			channels.ElevatorStates <- elevator
			channels.ElevatorStatesBroadcast <- elevator

		case <-doorTimer.C:
			if elevio.GetObstruction() {
				doorTimer.Reset(3 * time.Second)
				elevator.Obstr = true
				fmt.Println("Obstruction detected")
				channels.ElevatorStates <- elevator
				channels.ElevatorStatesBroadcast <- elevator
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
				motorErrorTimer.Reset(3 * time.Second)
			}
			channels.ElevatorStates <- elevator
			channels.ElevatorStatesBroadcast <- elevator
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
			channels.ElevatorStates <- elevator
			channels.ElevatorStatesBroadcast <- elevator
		//fullfør cab ordre
		case obstruction := <-drv_obstr:
			if !obstruction && elevator.Obstr {
				doorTimer.Reset(3 * time.Second)
				elevator.Obstr = false
				channels.ElevatorStates <- elevator
				channels.ElevatorStatesBroadcast <- elevator
			}

		}

	}
}
func updateLights(elevator *Elev) {
	for {
		for floor := 0; floor < N_FLOORS; floor++ {
			for button := 0; button < N_BUTTONS; button++ {
				elevio.SetButtonLamp(elevio.ButtonType(button), floor, elevator.Queue[floor][button])
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func saveLocalOrder(channels Channels, OrdersToQueoe chan Order) {
	var localOrders [N_FLOORS][N_BUTTONS][]Orderstatus
	for {
		select {
		case order := <-channels.OrderAssigned:
			//save order to local queue
			if order.Button == elevio.BT_Cab {
				OrdersToQueoe <- Order{order.Floor, order.Button}
				continue
			}
			fmt.Println("Order added to me: ")
			localOrders[order.Floor][order.Button] = append(localOrders[order.Floor][order.Button], order)
			OrdersToQueoe <- Order{order.Floor, order.Button}

		case order := <-channels.OrderComplete:
			fmt.Println("Order completed removing from local list")
			completedOrderList := localOrders[order.Floor][order.Button]
			//find order in queue based on floor and button
			localOrders[order.Floor][order.Button] = []Orderstatus{}
			for _, order := range completedOrderList {
				order.Status = true
				fmt.Println("Order completed deletet from list and sent to sherr or myself: ")

				//what if sherriff disconnect while sending this?=?!
				channels.OrderDelete <- order
			}
		}
	}

}
