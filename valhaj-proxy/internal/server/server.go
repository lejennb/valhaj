package server

import (
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"lj.com/valhaj-proxy/external/client/connection"
	"lj.com/valhaj-proxy/external/client/database"
	"lj.com/valhaj-proxy/external/client/reader"
	"lj.com/valhaj-proxy/internal/config"
)

type Server struct {
	listener net.Listener
	quit     chan bool
	WG       sync.WaitGroup
}

// NewServer(): Creates a new server instance.
func NewServer(network, address, caFile, certFile, keyFile string) *Server {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		log.Fatal(err)
	}

	// Load the CA certificate that signed the client certificate
	clientCA, err := os.ReadFile(caFile)
	if err != nil {
		log.Fatal(err)
	}

	certPool, _ := x509.SystemCertPool()
	if certPool == nil {
		certPool = x509.NewCertPool()
	}

	if !certPool.AppendCertsFromPEM(clientCA) {
		log.Fatal("failed to append the client CA certificate to the certificate pool")
	}

	config := tls.Config{
		ClientCAs:    certPool,
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		MinVersion:   tls.VersionTLS13,
		Rand:         rand.Reader,
	}
	listener, err := tls.Listen(network, address, &config)
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
			dbConn, err := connection.Connect(config.ServerDatabaseNetwork, config.ServerDatabaseAddress)
			if err != nil {
				log.Printf("Error: %s\n", err)
				syscall.Kill(syscall.Getpid(), syscall.SIGINT)
			} else {
				s.WG.Add(1)
				go func() {
					s.StartSession(conn, dbConn)
					s.WG.Done()
					if err := connection.Disconnect(dbConn); err != nil {
						log.Printf("Error: %s\n", err)
						syscall.Kill(syscall.Getpid(), syscall.SIGINT)
					}
				}()
			}
		}
	}
}

// StartSession(): Runs the client's session. Reads and executes commands and writes responses back to the client.
func (s *Server) StartSession(conn, dbConn net.Conn) {
	defer func() {
		conn.Close()
		if err := recover(); err != nil {
			log.Printf("Recovering from error: %s\n", err)
		}
	}()

	rd := reader.NewReader(conn)
	dbRd := reader.NewReader(dbConn) // TODO: What about '*.Reset()'?

SessionLoop:
	for {
		select {
		case <-s.quit:
			return
		default:
			conn.SetDeadline(time.Now().Add(config.ServerGracefulShutdownDelay * time.Millisecond))

			line, err := rd.Read()
			if err != nil {
				if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
					continue SessionLoop
				} else if err == io.EOF {
					return
				} else {
					_, _ = conn.Write([]uint8("!1\r\n" + "-ERR (PRX) " + err.Error() + "\r\n"))
					return
				}
			}

			responses, err := database.Exec(dbConn, dbRd, line)
			if err != nil { // TODO: Fully exit on error
				_, _ = conn.Write([]uint8("!1\r\n" + "-ERR (PRX) " + err.Error() + "\r\n"))
				return
			}
			rescount := len(responses)

			maxSize := rescount*2 + 3
			var fwResponses = make([]string, 0, maxSize)
			fwResponses = append(fwResponses, "!", strconv.Itoa(rescount), "\r\n")
			for _, response := range responses {
				fwResponses = append(fwResponses, response, "\r\n")
			}

			fwResponse := strings.Join(fwResponses, "")
			_, err = conn.Write([]uint8(fwResponse))
			if err != nil {
				log.Println(err)
				return
			}
		}
	}
}
