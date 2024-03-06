package deputy

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"mymodule/config"
	"mymodule/network/conn"
	. "mymodule/types"
	"net"
	"strings"
	"time"
)

var DeputyBecomeSheriff = make(chan map[string]Orderstatus)

func InitDeputy(networkchannels NetworkChannels) (bool, string) {
	sheriffIP := GetSheriffIP()
	dep2SherConn, err, sheriffID := connectDeputyToSheriff(sheriffIP)
	if err != nil {
		fmt.Println("Error connecting to sheriff:", err)
		return false, "nil"
	}

	go receiveDeputyMessage(dep2SherConn, networkchannels)
	return true, sheriffID
}

func connectDeputyToSheriff(sheriffIP string) (net.Conn, error, string) {
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", sheriffIP, config.Sheriff_deputy_port))
	if err != nil {
		fmt.Println("Error connecting to sheriff:", err)
		return nil, err, "nil"
	}
	reader := bufio.NewReader(conn)
	message, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Error reading from sheriff:", err)
		return nil, err, "nil"
	}
	fmt.Println("Received message from sheriff:", message)
	sheriffID := strings.TrimSpace(message)

	fmt.Println("Me, a deputy, connected to sheriff")

	return conn, nil, sheriffID
}

func receiveDeputyMessage(deputyToSheriffConn net.Conn, networkchannels NetworkChannels) {
	var deputyNodeOrders map[string]Orderstatus
	for {
		reader := bufio.NewReader(deputyToSheriffConn)
		message, err := reader.ReadString('\n')
		//fmt.Println("Received message from sheriff:", message)
		if err != nil {
			if err == io.EOF {
				fmt.Println("Connection closed from sheriff, deputy becomes sheriff")
				networkchannels.DeputyPromotion <- deputyNodeOrders
				return
			}
			fmt.Println("Error reading from sheriff:", err)
			time.Sleep(2 * time.Second)
			continue
		}

		var msg Message
		err = json.Unmarshal([]byte(message), &msg)
		if err != nil {
			fmt.Println("Error parsing message:", err)

			continue
		}

		switch msg.Type {
		case "deputyMessage":
			var orders map[string]Orderstatus
			err = json.Unmarshal(msg.Data, &orders)
			if err != nil {
				fmt.Println("Error parsing deputy message:", err)
				continue
			}
			deputyNodeOrders = orders
			// Handle deputy message...
			fmt.Println("Received deputy message from sheriff")

		default:
			fmt.Println("Unknown message type:", msg.Type)
		}
	}

}

func GetSheriffIP() string {
	var buf [1024]byte

	conn := conn.DialBroadcastUDP(config.Sheriff_port)
	defer conn.Close()
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	n, _, err := conn.ReadFrom(buf[0:])
	if err != nil {
		fmt.Println("Error reading from sheriff:", err)
		return ""
	}
	return string(buf[:n])

}
