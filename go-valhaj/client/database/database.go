package database

import (
	"errors"
	"net"
	"strconv"
	"strings"

	"lj.com/go-valhaj/client/reader"
)

var (
	errInvalidProtoCount = errors.New("invalid count protocol response format") // Cannot be empty (at least: "!X", len=2)
	errInvalidQueryCount = errors.New("invalid query count")

	countMinMessage = 2
)

// Exec(): Sends a query to the server for processing, returning the response in a series of *n* fragments.
func Exec(conn net.Conn, read *reader.Reader, query string) ([]string, error) {
	var empty []string
	var rescount int
	var resproto string
	var err error

	// Send query
	if _, err := conn.Write([]uint8(query + "\r\n")); err != nil {
		return empty, err
	}

	// Get response
	resproto, err = read.Read()
	if err != nil {
		return empty, err
	}
	if len(resproto) < countMinMessage {
		return empty, errInvalidProtoCount
	}

	rescount, err = strconv.Atoi(resproto[1:])
	if err != nil {
		return empty, err
	}

	var response = make([]string, 0, rescount)
	var fragment string
	for i := 0; i < rescount; i++ {
		fragment, err = read.Read()
		if err != nil {
			return empty, err
		}
		response = append(response, fragment)
	}

	return response, nil
}

// ExecPipeline(): Sends a series of queries to the server for processing, returning the responses in a series of *i* times *n* fragments.
func ExecPipeline(conn net.Conn, read *reader.Reader, queries []string) ([][]string, error) {
	var empty [][]string
	var cmdcount int
	var rescount int

	cmdcount = len(queries)
	if queries == nil || cmdcount == 0 {
		return empty, errInvalidQueryCount
	}

	// Assemble query
	pipelineQuery := strings.Join(queries, "\r\n")

	// Send query
	if _, err := conn.Write([]uint8(pipelineQuery + "\r\n")); err != nil {
		return empty, err
	}

	// Get responses
	var err error
	var resproto string
	var responses = make([][]string, 0, cmdcount)
	for i := 0; i < cmdcount; i++ {
		resproto, err = read.Read()
		if err != nil {
			return responses, err // Return the set of responses up until the error
		}
		if len(resproto) < countMinMessage {
			return responses, errInvalidProtoCount
		}

		rescount, err = strconv.Atoi(resproto[1:])
		if err != nil {
			return responses, err
		}

		var response = make([]string, 0, rescount)
		var fragment string
		for j := 0; j < rescount; j++ {
			fragment, err = read.Read()
			if err != nil {
				return responses, err
			}
			response = append(response, fragment)
		}

		responses = append(responses, response)
	}

	return responses, nil
}
