package localip

import (
	"fmt"
	"net"
	"strings"
	"time"
)

var localIP string

func LocalIP() string {
	for {
		if localIP == "" {
			conn, err := net.DialTCP("tcp4", nil, &net.TCPAddr{IP: []byte{8, 8, 8, 8}, Port: 53})
			if err != nil {
				if strings.Contains(err.Error(), "network is unreachable") {
					return "offline"
				}
				fmt.Println("Error getting IP from google DNS, retrying:", err)
				time.Sleep(3 * time.Second)
				continue
			}
			defer conn.Close()
			localIP = strings.Split(conn.LocalAddr().String(), ":")[0]
		}
		return localIP
	}
}
