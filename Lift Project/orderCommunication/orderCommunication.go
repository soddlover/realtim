package orderCom

import (
	"encoding/json"
	"fmt"
	"math/rand"
	elevatorFSM "mymodule/elevator"
	"mymodule/elevator/elevio"
	"mymodule/network"
	"mymodule/network/conn"
	"net"
	"reflect"
	"time"
)

type OrderCommunication struct {
	TransmitOrderChan chan OrderMessage
	ReiceveOrderChan  chan OrderMessage

	TransmitConfirmChan chan ConfirmMessage
	ReiceveConfirmChan  chan ConfirmMessage

	Port int
	IP   string //adress to the computer running this code
}

type OrderMessage struct {
	Key     string
	FromIP  string
	ToIP    string
	Payload network.OrderAndID
}

type ConfirmMessage struct {
	Key     string
	FromIP  string
	ToIP    string
	Payload network.OrderAndID
}

func NewOrderCommunication(channels elevatorFSM.Channels, id string) OrderCommunication {
	o := OrderCommunication{
		TransmitOrderChan:   make(chan OrderMessage),
		ReiceveOrderChan:    make(chan OrderMessage),
		TransmitConfirmChan: make(chan ConfirmMessage),
		ReiceveConfirmChan:  make(chan ConfirmMessage),

		Port: 15647, //// To be fix
		IP:   id,
	}

	go transmitter(o.Port, o.TransmitOrderChan, o.TransmitConfirmChan)
	go receiver(o.Port, o.ReiceveOrderChan, o.ReiceveConfirmChan)

	go o.confirmOrder(channels)

	return o
}

func (o OrderCommunication) sendOrder(order network.OrderAndID) bool {
	fmt.Println("Sending order...")
	randString := randomString(5)

	orderMessage := OrderMessage{
		Key:     randString,
		ToIP:    order.ID, // Assuming the IP is derived from order.ID; adjust as needed
		FromIP:  o.IP,
		Payload: order,
	}

	// Start the total timeout timer for 3 seconds
	timeout := time.After(3 * time.Second)
	attempts := 0

	for attempts < 3 {
		// Send the order
		o.TransmitOrderChan <- orderMessage

		select {
		case confirmation := <-o.ReiceveConfirmChan:
			if confirmation.Key == orderMessage.Key {
				fmt.Println("Order confirmed")
				return true // Order confirmed
			}
		case <-time.After(1 * time.Second): // Immediate retry without waiting for the full timeout
			attempts++
		case <-timeout:
			fmt.Println("Order confirmation timeout")
			return false // Total timeout reached
		}

	}
	fmt.Println("Order not confirmed after 3 attempts")
	return false // Order not confirmed after retries
}

func (o OrderCommunication) confirmOrder(channels elevatorFSM.Channels) {
	for {
		select {
		case m := <-o.ReiceveOrderChan:
			fmt.Println("Recieved an order")
			response := ConfirmMessage{
				Key:     m.Key,
				ToIP:    m.ToIP,
				FromIP:  m.FromIP,
				Payload: m.Payload,
			}

			fmt.Print("Comfirmed order") //It will now accept every order
			o.TransmitConfirmChan <- response

			channels.OrderAssigned <- m.Payload.Order
		}
	}

}

const bufSize = 1024

func transmitter(port int, chans ...interface{}) {

	checkArgs(chans...)
	typeNames := make([]string, len(chans))
	selectCases := make([]reflect.SelectCase, len(typeNames))
	for i, ch := range chans {
		selectCases[i] = reflect.SelectCase{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(ch),
		}
		typeNames[i] = reflect.TypeOf(ch).Elem().String()
	}

	for {

		chosen, value, _ := reflect.Select(selectCases)
		jsonstr, _ := json.Marshal(value.Interface())
		ttj, _ := json.Marshal(typeTaggedJSON{
			TypeId: typeNames[chosen],
			JSON:   jsonstr,
		})
		if len(ttj) > bufSize {
			panic(fmt.Sprintf(
				"Tried to send a message longer than the buffer size (length: %d, buffer size: %d)\n\t'%s'\n"+
					"Either send smaller packets, or go to network/bcast/bcast.go and increase the buffer size",
				len(ttj), bufSize, string(ttj)))
		}

		// Unmarshal ttj back into typeTaggedJSON struct to access the JSON field
		var wrapped typeTaggedJSON
		json.Unmarshal(ttj, &wrapped)

		// Unmarshal the JSON field into a map to access the ToIP field
		var originalObject map[string]interface{}
		json.Unmarshal(wrapped.JSON, &originalObject)

		// Extract the ToIP field
		toIP := originalObject["ToIP"].(string)
		fmt.Println(toIP)

		conn := conn.DialBroadcastUDP(port)
		addr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("%s:%d", toIP, port))
		fmt.Println(err)
		conn.WriteTo(ttj, addr)

	}
}

func receiver(port int, chans ...interface{}) {
	checkArgs(chans...)
	chansMap := make(map[string]interface{})
	for _, ch := range chans {
		chansMap[reflect.TypeOf(ch).Elem().String()] = ch
	}

	var buf [bufSize]byte
	conn := conn.DialBroadcastUDP(port)
	for {
		n, _, e := conn.ReadFrom(buf[0:])
		if e != nil {
			fmt.Printf("bcast.Receiver(%d, ...):ReadFrom() failed: \"%+v\"\n", port, e)
		}

		var ttj typeTaggedJSON
		json.Unmarshal(buf[0:n], &ttj)
		ch, ok := chansMap[ttj.TypeId]
		if !ok {
			continue
		}
		v := reflect.New(reflect.TypeOf(ch).Elem())
		json.Unmarshal(ttj.JSON, v.Interface())
		reflect.Select([]reflect.SelectCase{{
			Dir:  reflect.SelectSend,
			Chan: reflect.ValueOf(ch),
			Send: reflect.Indirect(v),
		}})
	}
}

type typeTaggedJSON struct {
	TypeId string
	JSON   []byte
}

// Checks that args to Tx'er/Rx'er are valid:
//
//	All args must be channels
//	Element types of channels must be encodable with JSON
//	No element types are repeated
//
// Implementation note:
//   - Why there is no `isMarshalable()` function in encoding/json is a mystery,
//     so the tests on element type are hand-copied from `encoding/json/encode.go`
func checkArgs(chans ...interface{}) {
	n := 0
	for range chans {
		n++
	}
	elemTypes := make([]reflect.Type, n)

	for i, ch := range chans {
		// Must be a channel
		if reflect.ValueOf(ch).Kind() != reflect.Chan {
			panic(fmt.Sprintf(
				"Argument must be a channel, got '%s' instead (arg# %d)",
				reflect.TypeOf(ch).String(), i+1))
		}

		elemType := reflect.TypeOf(ch).Elem()

		// Element type must not be repeated
		for j, e := range elemTypes {
			if e == elemType {
				panic(fmt.Sprintf(
					"All channels must have mutually different element types, arg# %d and arg# %d both have element type '%s'",
					j+1, i+1, e.String()))
			}
		}
		elemTypes[i] = elemType

		// Element type must be encodable with JSON
		checkTypeRecursive(elemType, []int{i + 1})

	}
}

func checkTypeRecursive(val reflect.Type, offsets []int) {
	switch val.Kind() {
	case reflect.Complex64, reflect.Complex128, reflect.Chan, reflect.Func, reflect.UnsafePointer:
		panic(fmt.Sprintf(
			"Channel element type must be supported by JSON, got '%s' instead (nested arg# %v)",
			val.String(), offsets))
	case reflect.Map:
		if val.Key().Kind() != reflect.String {
			panic(fmt.Sprintf(
				"Channel element type must be supported by JSON, got '%s' instead (map keys must be 'string') (nested arg# %v)",
				val.String(), offsets))
		}
		checkTypeRecursive(val.Elem(), offsets)
	case reflect.Array, reflect.Ptr, reflect.Slice:
		checkTypeRecursive(val.Elem(), offsets)
	case reflect.Struct:
		for idx := 0; idx < val.NumField(); idx++ {
			checkTypeRecursive(val.Field(idx).Type, append(offsets, idx+1))
		}
	}
}

func randomString(n int) string {
	letters := "abc123"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func TestOrderCommunication(channels elevatorFSM.Channels, world network.World, id string) {

	ordercom := NewOrderCommunication(channels, id)

	new_order := elevatorFSM.Order{
		Floor:  2,
		Button: elevio.BT_HallUp,
	}

	order_and_id := network.OrderAndID{
		Order: new_order,
		ID:    id,
	}

	ordercom.sendOrder(order_and_id)

}
