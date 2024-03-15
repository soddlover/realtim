package localip

import (
	"fmt"
	"net"
	"strings"
	"time"
)

var localIP string

func LocalIP() string {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			return "offline"
		default:
			if localIP == "" {
				conn, err := net.DialTCP("tcp4", nil, &net.TCPAddr{IP: []byte{8, 8, 8, 8}, Port: 53})
				if err != nil {
					if strings.Contains(err.Error(), "network is unreachable") {
						fmt.Println("Network is unreachable, retrying:", err)
						continue
					}
					fmt.Println("Error getting IP from google DNS, retrying:", err)
					continue
				}
				defer conn.Close()
				localIP = strings.Split(conn.LocalAddr().String(), ":")[0]
			}
			return localIP
		}
	}
}
