package main

import (
	"fmt"
	"log"
	"time"

	"lj.com/go-valhaj/client/connection"
	"lj.com/go-valhaj/client/database"
	"lj.com/go-valhaj/client/reader"
)

func main() {
	conn, err := connection.Connect("tcp", "127.0.0.1:6380")
	if err != nil {
		log.Fatalf("error: %s", err)
	}

	read := reader.NewReader(conn)

	// Assemble some queries
	base64Image := `iVBORw0KGgoAAAANSUhEUgAAAA8AAAAYCAIAAACqZzA+AAAAA3NCSVQICAjb4U/gAAAAGXRFWHRTb2Z0d2FyZQBnbm9tZS1zY3JlZW5zaG907wO/PgAAAYxJREFUOI1Vk0GC5DYMAwuQ9xl7yf+f2GLlQPdM4pMti1ARhPL3n4eJKoXCADCqmmT/Jqfy3HtjARHuGLlJYiAM9FRmRnmkSVgZgGmaJDIfhQBNqfpAZzznufe2jQScIVmqWTTzVgCL+PNOMzMGGmC57tiTzszMXMcASK/5kP3cjhkjVdsaVlujzkxb9TN3K4jEZxCKLEaW6ig3NiBZ/bYFZmZx9zlt/Sq+RrE7H7UhtUEtZytLWmbut6bqkxgaWbmXJ1Hn/0apL4nKXOYlcII1DBrWnCRVdxVI1NeElQSwTtKOPh9tO3qCGgLTIDoh+SzbzEl6zrMyk0L3XLPTeN24jnrxyc3hXJyxZwc04RjVJuCfV9FnAQuTzJUIDG/6vobMftXMdxKzyd5oFA6FfG2Y6zz/bf+cs3P92hyAJt+Vh9O1uYIe9l4JnYSfPvZWfO6dkGTe2P8G5t57nWXbPf3dkfMZhg7FJmm7W5M4XHnUwOSnfSAXnfEbQEFJ0nLeLhs2z5uC8x47+fEs/wLXb539ywuOGAAAAABJRU5ErkJggg==` // base64 --wrap=0 test.png
	queries := []string{
		`SET 0001 "\"Hello, world!\n\""`,
		"GET 0001",
		"SET 0002 3847849748947897893748937483947389485648957857648974893743847894",
		"GET 0002",
		"EXISTS 0001 0002",
		"DEL 0001",
		`SET 0003 "{\"data\":\"hello, world!\n\"}"`,
		fmt.Sprintf(`SET 0004 "%s"`, base64Image),
		"QUIT",
	}

	start := time.Now()
	for _, query := range queries {
		if res, err := database.Exec(conn, read, query); err != nil {
			fmt.Printf("%s\n", err)
		} else {
			fmt.Printf("%v\n", res)
		}
	}

	read.Reset()

	if err := connection.Disconnect(conn); err != nil {
		log.Fatalf("error: %s", err)
	}

	duration := time.Since(start)
	fmt.Printf("%0.12fs\n", duration.Seconds())
}
