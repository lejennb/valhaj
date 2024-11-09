package reader

import (
	"bufio"
	"errors"
	"net"

	"lj.com/valhaj/internal/commands"
)

var (
	errIncompleteEmptyData = errors.New("incomplete or empty client data stream")
	errIncongruousQuotes   = errors.New("incongruous quotes")

	readEmptyMessage = 2 // Empty messages are fine (can be: "\r\n", len=2)
	readMinMessage   = 3 // Don't tolerate empty messages (at least: "X\r\n", len=3)
)

// Reader contains the logic to read from a raw tcp connection and create commands.
type Reader struct {
	conn net.Conn
	br   *bufio.Reader
	line []byte
	pos  int
}

// NewReader(): Returns a new Reader that reads from the given connection.
func NewReader(conn net.Conn) *Reader {
	return &Reader{
		conn: conn,
		br:   bufio.NewReader(conn),
		line: make([]byte, 0),
		pos:  0,
	}
}

func (r *Reader) current() byte {
	if r.end() {
		return '\n'
	}
	return r.line[r.pos]
}

func (r *Reader) advance() {
	r.pos++
}

func (r *Reader) end() bool {
	return r.pos >= len(r.line)
}

// consumeString(): Reads a string argument from the current line.
func (r *Reader) consumeString() (s []byte, err error) {
	var escape byte = '\\'
	for r.current() != '"' && !r.end() {
		cur := r.current()
		r.advance()
		if cur == escape {
			next := r.current()
			s = append(s, cur, next)
			r.advance()
		} else {
			s = append(s, cur)
		}
	}
	if r.current() != '"' {
		return nil, errIncongruousQuotes
	}
	r.advance()
	return
}

// consumeArg(): Reads an argument from the current line.
func (r *Reader) consumeArg() (s string, err error) {
	for r.current() == ' ' {
		r.advance()
	}
	if r.current() == '"' {
		r.advance()
		buf, err := r.consumeString()
		return string(buf), err
	}
	for !r.end() && r.current() != ' ' && r.current() != '\n' {
		s += string(r.current())
		r.advance()
	}
	return
}

func (r *Reader) readLine() ([]byte, error) {
	line, err := r.br.ReadBytes('\n')
	if err != nil {
		return nil, err
	}

	// Check for complete, "\r\n" terminated data stream
	lineLen := len(line)
	if lineLen < readMinMessage {
		return nil, errIncompleteEmptyData
	}
	if line[lineLen-2] != '\r' {
		return nil, errIncompleteEmptyData
	}
	return line[:lineLen-2], nil
}

// Read(): Reads and returns a commands.Command.
func (r *Reader) Read() (commands.Command, error) {
	line, err := r.readLine()
	if err != nil {
		return commands.Command{}, err
	}
	r.pos = 0
	r.line = []byte{}
	r.line = append(r.line, line...)

	// Skip initial whitespace if any
	for r.current() == ' ' {
		r.advance()
	}
	cmd := commands.Command{Connection: r.conn}
	for !r.end() {
		arg, err := r.consumeArg()
		if err != nil {
			return cmd, err
		}
		if arg != "" {
			cmd.Arguments = append(cmd.Arguments, arg)
		}
	}
	return cmd, nil
}
