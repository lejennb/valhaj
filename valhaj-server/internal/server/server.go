package server

import (
	"io"
	"log"
	"net"
	"sync"
	"time"

	"lj.com/valhaj/internal/config"
	"lj.com/valhaj/internal/memory"
	"lj.com/valhaj/internal/reader"
	"lj.com/valhaj/internal/writer"
)

type Server struct {
	listener net.Listener
	quit     chan bool
	WG       sync.WaitGroup
}

// NewServer(): Creates a new server instance.
func NewServer(network, address string) *Server {
	listener, err := net.Listen(network, address)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Listening on %s: %s\n", network, address)

	s := &Server{
		listener: listener,
		quit:     make(chan bool),
	}
	return s
}

// Quit(): Shuts down the server instance.
func (s *Server) Quit() {
	close(s.quit)
	log.Println("Closing listener")
	s.listener.Close()
	s.WG.Wait()
}

// Serve(): Launches the server instance.
func (s *Server) Serve() {
	defer s.WG.Done()

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.quit:
				return
			default:
				log.Printf("Error: %s\n", err)
			}
		} else {
			s.WG.Add(1)
			go func() {
				s.StartSession(conn)
				s.WG.Done()
			}()
		}
	}
}

// StartSession(): Runs the client's session. Reads and executes commands and writes responses back to the client.
func (s *Server) StartSession(conn net.Conn) {
	var oldIndex = -1
	var newIndex = 0
	var database memory.ShardedCache
	var status bool

	defer func() {
		conn.Close()
	}()
	defer func() {
		if err := recover(); err != nil {
			log.Printf("Recovering from error: %s\n", err)
		}
	}()

	r := reader.NewReader(conn)

SessionLoop:
	for {
		select {
		case <-s.quit:
			return
		default:
			conn.SetDeadline(time.Now().Add(config.ServerGracefulShutdownDelay * time.Millisecond))
			cmd, err := r.Read()
			if err != nil {
				if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
					continue SessionLoop
				} else if err == io.EOF {
					return
				} else {
					// We're closing the client connection due to other errors, no need to handle write errors
					responses := []string{"!1\r\n", "-ERR ", err.Error(), "\r\n"}
					_, _ = conn.Write(writer.BuildResponse(responses))
					return
				}
			}

			if cmd.Empty() {
				responses := []string{"!1\r\n", "-ERR superfluous write\r\n"}
				_, _ = conn.Write(writer.BuildResponse(responses))
				return
			}

			if newIndex != oldIndex { // Only update the DB reference if the index changed
				database = *memory.Container[newIndex]
				oldIndex = newIndex
			}
			cmd.Index = newIndex
			cmd.Database = database

			newIndex, status = cmd.Execute()
			if !status {
				return
			}
		}
	}
}
