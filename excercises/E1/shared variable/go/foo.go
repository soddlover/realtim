// Use `go run foo.go` to run your program

package main

import (
	. "fmt"
	"runtime"
)

var i = 0

func server(inc chan bool, dec chan bool, quit chan bool, done chan bool) {
	for {
		select {
		case <-inc:
			i++
		case <-dec:
			i--
		case <-quit:
			Println("The magic number is:", i)
			done <- true
		}
	}
}

func incrementing(inc chan bool, done chan bool) {
	//TODO: increment i 1000000 times
	for j := 0; j < 1000001; j++ {
		inc <- true
	}
	done <- true
}

func decrementing(dec chan bool, done chan bool) {
	//TODO: decrement i 1000000 times
	for j := 0; j < 1000000; j++ {
		dec <- true
	}
	done <- true
}

func main() {
	// What does GOMAXPROCS do? What happens if you set it to 1?
	runtime.GOMAXPROCS(3)
	inc := make(chan bool)
	dec := make(chan bool)
	done := make(chan bool)
	quit := make(chan bool)
	// TODO: Spawn both functions as goroutines
	go server(inc, dec, quit, done)
	go decrementing(dec, done)
	go incrementing(inc, done)

	<-done
	<-done
	quit <- true
	<-done

	// We have no direct way to wait for the completion of a goroutine (without additional synchronization of some sort)
	// We will do it properly with channels soon. For now: Sleep.
}
