package main

import (
	"fmt"
	"log"
)

func main() {
	log.Printf("Test")

	poruke := make(chan string)

	go func() {
		poruke <- "Pinguj!"
	}()

	msg := <-poruke
	fmt.Println(msg)
}
