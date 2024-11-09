package connection

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net"
	"os"
)

// Connect(): Opens a new unencrypted connection to the server.
func Connect(network, address string) (net.Conn, error) {
	conn, err := net.Dial(network, address)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

// ConnectTLS(): Opens a new encrypted connection based on mTLS authentication.
func ConnectTLS(network, address, caFile, certFile, keyFile string) (net.Conn, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}

	serverCA, err := os.ReadFile(caFile)
	if err != nil {
		return nil, err
	}

	certPool, _ := x509.SystemCertPool()
	if certPool == nil {
		certPool = x509.NewCertPool()
	}

	if !certPool.AppendCertsFromPEM(serverCA) {
		return nil, errors.New("failed to append the server CA certificate to the certificate pool")
	}

	config := tls.Config{
		RootCAs:            certPool,
		Certificates:       []tls.Certificate{cert},
		InsecureSkipVerify: false,
	}
	conn, err := tls.Dial(network, address, &config)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

// Disconnect(): Closes a given connection to the server.
func Disconnect(conn net.Conn) error {
	if err := conn.Close(); err != nil {
		return err
	}
	return nil
}
