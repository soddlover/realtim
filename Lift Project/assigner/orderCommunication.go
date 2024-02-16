package assigner

import (
	"encoding/json"
	"fmt"
	"math/rand"
	elevatorFSM "mymodule/elevator"
	"net"
	"reflect"
	"time"
)

type OrderCommunication struct {
	TransmitOrderChan chan OrderMessage
	ReiceveOrderChan  chan OrderMessage

	TransmitConfirmChan chan OrderMessage
	ReiceveConfirmChan  chan OrderMessage

	Port     int
	IPOfThis string
}

type OrderMessage struct {
	Key       string
	IP        string
	IP_from   string
	Payload   OrderAndID
	Confirmed bool
}

func NewOrderCommunication(channels elevatorFSM.Channels, id string) OrderCommunication {
	o := OrderCommunication{
		TransmitOrderChan:   make(chan OrderMessage),
		ReiceveOrderChan:    make(chan OrderMessage),
		TransmitConfirmChan: make(chan OrderMessage),
		ReiceveConfirmChan:  make(chan OrderMessage),

		Port:     15647, //// To be fix
		IPOfThis: id,
	}

	go transmitter(o.Port, o.TransmitOrderChan, o.TransmitConfirmChan)
	go receiver(o.Port, o.ReiceveOrderChan, o.ReiceveConfirmChan)

	go o.confirmOrder(channels)

	return o
}

func (o OrderCommunication) sendOrder(order OrderAndID) bool {
	fmt.Println("Sending order...")
	randString := randomString(5)

	orderMessage := OrderMessage{
		Key:       randString,
		IP:        order.ID, // Assuming the IP is derived from order.ID; adjust as needed
		IP_from:   o.IPOfThis,
		Payload:   order,
		Confirmed: false,
	}

	// Start the total timeout timer for 3 seconds
	timeout := time.After(3 * time.Second)
	attempts := 0

	for attempts < 3 {
		// Send the order
		o.TransmitOrderChan <- orderMessage

		select {
		case confirmation := <-o.ReiceveConfirmChan:
			if confirmation.Key == orderMessage.Key && confirmation.Confirmed {
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
			response := OrderMessage{
				Key:       m.Key,
				IP:        m.IP_from,
				IP_from:   m.IP,
				Payload:   m.Payload,
				Confirmed: true,
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
	selectCases := make([]reflect.SelectCase, len(chans))
	for i, ch := range chans {
		selectCases[i] = reflect.SelectCase{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(ch),
		}
		typeNames[i] = reflect.TypeOf(ch).Elem().String()
	}

	for {
		chosen, value, ok := reflect.Select(selectCases)
		if !ok {
			continue // Channel closed or other error, handle appropriately
		}

		// Type assert to OrderMessage
		orderMsg, isOrderMsg := value.Interface().(OrderMessage)
		if !isOrderMsg {
			fmt.Println("Received value is not an OrderMessage")
			continue // Or handle the error as necessary
		}

		// Extract IP and marshal the payload
		ipAddr := orderMsg.IP
		addr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("%s:%d", ipAddr, port))
		if err != nil {
			fmt.Printf("Error resolving UDP address: %v\n", err)
			continue
		}

		conn, err := net.DialUDP("udp", nil, addr)
		if err != nil {
			fmt.Printf("Error dialing UDP: %v\n", err)
			continue
		}
		defer conn.Close()

		jsonstr, err := json.Marshal(orderMsg.Payload)
		if err != nil {
			fmt.Printf("Error marshaling payload: %v\n", err)
			continue
		}

		ttj, err := json.Marshal(typeTaggedJSON{
			TypeId: typeNames[chosen],
			JSON:   jsonstr,
		})
		if err != nil {
			fmt.Printf("Error marshaling typeTaggedJSON: %v\n", err)
			continue
		}

		if len(ttj) > bufSize {
			fmt.Printf("Message longer than buffer size (length: %d, buffer size: %d)\n", len(ttj), bufSize)
			continue // Or handle the error as necessary
		}

		_, err = conn.Write(ttj)
		if err != nil {
			fmt.Printf("Error sending data: %v\n", err)
		}
	}
}

// Matches type-tagged JSON received on `port` to element types of `chans`, then
// sends the decoded value on the corresponding channel
func receiver(port int, chans ...interface{}) {
	checkArgs(chans...)
	chansMap := make(map[string]reflect.Value)
	for _, ch := range chans {
		chansMap[reflect.TypeOf(ch).Elem().String()] = reflect.ValueOf(ch)
	}

	addr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf(":%d", port))
	if err != nil {
		fmt.Printf("Error resolving UDP address: %v\n", err)
		return
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		fmt.Printf("Error listening on UDP: %v\n", err)
		return
	}
	defer conn.Close()

	var buf [bufSize]byte
	for {
		n, _, err := conn.ReadFromUDP(buf[:])
		if err != nil {
			fmt.Printf("Receiver(%d, ...): ReadFromUDP() failed: \"%+v\"\n", port, err)
			continue
		}

		var ttj typeTaggedJSON
		if err := json.Unmarshal(buf[:n], &ttj); err != nil {
			fmt.Printf("Error unmarshalling JSON: %v\n", err)
			continue
		}

		ch, ok := chansMap[ttj.TypeId]
		if !ok {
			fmt.Printf("No channel found for type: %s\n", ttj.TypeId)
			continue
		}

		v := reflect.New(reflect.TypeOf(ch.Interface()).Elem())
		if err := json.Unmarshal(ttj.JSON, v.Interface()); err != nil {
			fmt.Printf("Error unmarshalling to type: %v\n", err)
			continue
		}

		reflect.Select([]reflect.SelectCase{{
			Dir:  reflect.SelectSend,
			Chan: ch,
			Send: v.Elem(),
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
