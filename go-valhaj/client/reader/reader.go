package reader

import (
	"bufio"
	"errors"
	"net"
)

var (
	errIncompleteEmptyData = errors.New("incomplete or empty server data stream")

	readEmptyMessage = 2
)

type Reader struct {
	conn net.Conn
	br   *bufio.Reader
}

func NewReader(conn net.Conn) *Reader {
	return &Reader{
		conn: conn,
		br:   bufio.NewReader(conn),
	}
}

func (r *Reader) Read() (string, error) {
	line, err := r.br.ReadString('\n')
	if err != nil {
		return "", err
	}

	// Check for complete, "\r\n" terminated data stream
	lineLen := len(line)
	if lineLen < readEmptyMessage {
		return "", errIncompleteEmptyData
	}
	if line[lineLen-2] != '\r' {
		return "", errIncompleteEmptyData
	}
	return line[:lineLen-2], nil
}

func (r *Reader) Reset() {
	r.br.Reset(r.conn)
}
