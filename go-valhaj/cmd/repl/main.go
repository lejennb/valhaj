package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"lj.com/go-valhaj/client/connection"
	"lj.com/go-valhaj/client/database"
	"lj.com/go-valhaj/client/reader"
)

func main() {
	caFile, certFile, keyFile := "./ca-cert.pem", "./client-cert.pem", "./client-key.pem"
	network, address := "tcp", "127.0.0.1:6380"
	fmt.Printf("Trying %s via %s\n", address, network)

	conn, err := connection.ConnectTLS(network, address, caFile, certFile, keyFile)
	if err != nil {
		log.Fatalf("error: %s", err)
	}

	read := reader.NewReader(conn)

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		query := scanner.Text()
		if responses, err := database.Exec(conn, read, query); err != nil {
			fmt.Printf("%s\n", err)
			break
		} else {
			for _, response := range responses {
				fmt.Printf("%s\n", response)
			}
		}

		if strings.ToUpper(query) == "QUIT" {
			break
		}
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}

	read.Reset()

	if err := connection.Disconnect(conn); err != nil {
		log.Fatalf("error: %s", err)
	}
}
