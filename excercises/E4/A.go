/*
- `A`: Make a program that just terminates after a few seconds
- `B`: Make a program that starts program `A` in a separate window
- `B`: After starting `A`, send/write some data periodically
- `A`: Instead of just waiting, read data that `B` creates
- `A`: Add some kind of timeout to detect when `B` dies
*/
package main

import (
    "fmt"
    "net"
    "time"
	"os/exec"
	"log"
)

func main() {
    pc, err := net.ListenPacket("udp", ":8080")
    if err != nil {
        fmt.Println(err)
        return
    }
    defer pc.Close()

    buffer := make([]byte, 1024)
    fmt.Println("--- Backup mode---")
	lastReceivedNumber:=0
    for {
        pc.SetReadDeadline(time.Now().Add(2 * time.Second)) // Timeout if no data is received for 10 seconds
        _, _, err := pc.ReadFrom(buffer)
        if err != nil {
            fmt.Println("Timeout. No data received for 3 secondd, TAKING OVER.")

			break
        }
		receivedNumber := int(buffer[0]) // Assuming the received number is a single byte
		lastReceivedNumber=receivedNumber
		fmt.Println("recieved something")
    }
	pc.Close()
	fmt.Println("opening new terminal")
	cmd:=exec.Command("gnome-terminal","--","./A")
	err=cmd.Run()
	if err!=nil{
		fmt.Println("THis sucks")
		log.Fatal(err)
	}

	conn, err := net.Dial("udp", "localhost:8080")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer conn.Close()

	fmt.Println("SENDING SHIT")
	ticker:=time.NewTicker(1*time.Second)
	defer ticker.Stop()
	counter:=lastReceivedNumber
	
    for range ticker.C {
        counter++
        msg :=byte(counter)
        fmt.Printf("Sending: %d\n", msg)
        _, err := conn.Write([]byte{msg})
        if err != nil {
            fmt.Println(err)
            return
        }
		if counter >=(lastReceivedNumber+5){
			fmt.Printf("Program crashing oh no :(")
			return
		}
    }

}