// TODO: This is only an idea, valhaj-benchmark should be way more functional!
package main

import (
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"lj.com/valhaj-benchmark/external/client/connection"
	"lj.com/valhaj-benchmark/external/client/database"
	"lj.com/valhaj-benchmark/external/client/reader"
)

func main() { // Start N clients that each run 1000000/N * 2 commands (2000000)
	clients := 16
	iterations := 1000000 / clients

	var offset, index int
	var key string

	var wg sync.WaitGroup
	wg.Add(clients)

	start := time.Now()
	for i := 0; i < clients; i++ {
		offset = i * iterations
		go func(offset int) {
			defer wg.Done()
			conn, err := connection.Connect("tcp", "127.0.0.1:6380")
			if err != nil {
				log.Fatalf("error: %s", err)
			}

			read := reader.NewReader(conn)

			for j := 0; j < iterations; j++ {
				index = offset + j
				key = strconv.Itoa(index)
				database.Exec(conn, read, "SET "+key+" hello") // TODO: Easily switch benchmarked commands (higher-order func?)
				database.Exec(conn, read, "GET "+key)
			}
			database.Exec(conn, read, "QUIT")

			read.Reset()

			if err := connection.Disconnect(conn); err != nil {
				log.Fatalf("error: %s", err)
			}
		}(offset)
	}

	wg.Wait()

	duration := time.Since(start)
	fmt.Printf("%0.12fs\n", duration.Seconds())
}
