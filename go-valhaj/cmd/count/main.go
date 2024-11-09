package main

import (
	"fmt"
	"log"
	"sync"
	"time"

	"lj.com/go-valhaj/client/connection"
	"lj.com/go-valhaj/client/database"
	"lj.com/go-valhaj/client/reader"
)

func runClient() {
	conn, err := connection.Connect("tcp", "127.0.0.1:6380")
	if err != nil {
		log.Fatalf("error: %s", err)
	}

	read := reader.NewReader(conn)

	for i := 0; i < 500; i++ {
		database.Exec(conn, read, "INCR 1024")
	}
	database.Exec(conn, read, "QUIT")

	read.Reset()

	if err := connection.Disconnect(conn); err != nil {
		log.Fatalf("error: %s", err)
	}
}

func main() {
	clients := 16

	var wg sync.WaitGroup
	wg.Add(clients)

	start := time.Now()
	for i := 0; i < clients; i++ {
		go func() {
			defer wg.Done()

			runClient()
		}()
	}

	wg.Wait()

	duration := time.Since(start)
	fmt.Printf("%0.12fs\n", duration.Seconds())
}
