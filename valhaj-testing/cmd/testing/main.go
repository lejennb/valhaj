package main

import (
	"log"
	"net"
	"slices"
	"time"

	"lj.com/valhaj-testing/external/client/connection"
	"lj.com/valhaj-testing/external/client/database"
	"lj.com/valhaj-testing/external/client/reader"
)

var TotalTests int
var TotalCommands, PassedCommands, FailedCommands int
var TotalAsserts, PassedAsserts, FailedAsserts int
var Conn net.Conn
var Read *reader.Reader

func main() {
	start := time.Now()

	// Run the timed test suite
	RunTests()

	duration := time.Since(start)
	log.Printf("Done, took %0.12fs.\n", duration.Seconds())

	// Get the summary
	Summary()
}

func Summary() {
	log.Printf("Ran \x1b[1m%d tests\x1b[0m.\n", TotalTests)
	log.Printf("Ran \x1b[96m%d commands\x1b[39m: \x1b[32m%d passed\x1b[39m and \x1b[91m%d failed\x1b[39m.\n", TotalCommands, PassedCommands, FailedCommands)
	log.Printf("Ran \x1b[95m%d asserts\x1b[39m: \x1b[32m%d passed\x1b[39m and \x1b[91m%d failed\x1b[39m.\n", TotalAsserts, PassedAsserts, FailedAsserts)
}

// Context(): Visually separate tests from each other.
func Context(context string) {
	TotalTests++
	log.Printf("\x1b[33mContext: Running tests for the '%s' command.\x1b[39m\n", context)
}

// Eval(): Directly tests a command by observing its output and matching it against a full/partial set of expected values.
func Eval(command string, expectedOutput []string, contains bool) {
	TotalCommands++

	if len(expectedOutput) == 0 {
		log.Printf("\x1b[91mFail: Command '%s'. Expected output cannot be empty!\x1b[39m", command)
		return
	}

	responses, err := database.Exec(Conn, Read, command)

	var comparison bool
	if contains { // We only require one item to match our expected result
		comparison = slices.Contains(responses, expectedOutput[0])
	} else { // We require the full slice to match our expected result
		if slices.Compare(responses, expectedOutput) != 0 {
			comparison = false
		} else {
			comparison = true
		}
	}

	if !comparison || err != nil {
		FailedCommands++
		log.Printf("\x1b[91mFail: Command '%s'. Error: '%v'. Got %v, but expected %v.\x1b[39m", command, err, responses, expectedOutput)
	} else {
		PassedCommands++
		log.Printf("\x1b[32mPass: Command '%s'.\x1b[39m", command)
	}
}

// Assert(): Asserts do not test commands directly, but rather the the side-effects, or changes, following the execution of the command.
func Assert(command string, expectedOutput []string, contains bool) {
	TotalAsserts++

	if len(expectedOutput) == 0 {
		log.Printf("\x1b[90mFail: Assert '%s'. Expected output cannot be empty!\x1b[39m", command)
		return
	}

	responses, err := database.Exec(Conn, Read, command)

	var comparison bool
	if contains { // We only require one item to match our expected result
		comparison = slices.Contains(responses, expectedOutput[0])
	} else { // We require the full slice to match our expected result
		if slices.Compare(responses, expectedOutput) != 0 {
			comparison = false
		} else {
			comparison = true
		}
	}

	if !comparison || err != nil {
		FailedAsserts++
		log.Printf("\x1b[90mFail: Assert '%s'. Error: '%v'. Got %v, but expected %v.\x1b[39m", command, err, responses, expectedOutput)
	} else {
		PassedAsserts++
		log.Printf("\x1b[90mPass: Assert '%s'.\x1b[39m", command)
	}
}

// Setup(): Used to issue commands that modify - or prepare - the testing environment, e.g. by providing data to manipulate.
func Setup(command string) {
	_, _ = database.Exec(Conn, Read, command)
	log.Printf("\x1b[90mSetup: '%s'.\x1b[39m", command)
}

func RunTests() {
	/* Setup */
	var err error
	Conn, err = connection.Connect("tcp", "127.0.0.1:6380")
	if err != nil {
		log.Fatalf("error: %s", err)
	}
	Read = reader.NewReader(Conn)

	/* Commands */
	Context("select")
	Eval("select 0", []string{"+OK"}, false)
	Eval("select 100", []string{"-ERR index value is out of bounds"}, false)
	Eval("select", []string{"-ERR wrong number of arguments for 'select' command"}, false)
	Eval("select abc", []string{"-ERR index value is not an integer"}, false)

	Context("flushall")
	Eval("flushall", []string{"+OK"}, false)
	Assert("info", []string{"memory_logical_databases:3"}, true) // Just to make sure we got 3 DB's
	Assert("info", []string{"keyspace_keys:0"}, true)
	Setup("select 1")
	Assert("info", []string{"keyspace_keys:0"}, true)
	Setup("select 2")
	Assert("info", []string{"keyspace_keys:0"}, true)
	Setup("select 0") // Let's switch back

	Context("move")
	Eval("move 454545 1", []string{"-ERR no such key"}, false)
	Setup("set 454545 hello")
	Eval("move 454545 1", []string{"+OK"}, false)
	Setup("set 454545 hello")
	Eval("move 454545 1", []string{"-ERR key already exists in destination database"}, false)
	Eval("move 454545 0", []string{"+OK"}, false)

	Context("mset")
	Eval("mset 500 hi 600 bye", []string{"+OK"}, false)
	Eval("mset 500 hi 600 bye 700", []string{"-ERR wrong number of arguments for 'mset' command"}, false)
	Eval("mset 500 hi 600", []string{"-ERR wrong number of arguments for 'mset' command"}, false)

	Context("mget")
	Eval("mget 500 600", []string{"hi", "bye"}, false)
	Eval("mget 800 900", []string{"", ""}, false)
	Eval("mget", []string{"-ERR wrong number of arguments for 'mget' command"}, false)

	Context("get")
	Eval("get 600", []string{"bye"}, false)
	Eval("get 900", []string{""}, false)
	Eval("get", []string{"-ERR wrong number of arguments for 'get' command"}, false)

	Context("set")
	Eval("set 600 hi", []string{"+OK"}, false)
	Eval("set 600 hi ???", []string{"-ERR wrong syntax for 'set' command"}, false)
	Eval("set 600", []string{"-ERR wrong number of arguments for 'set' command"}, false)
	Eval("set 600 hi nx", []string{""}, false)
	Eval("set 600 hi nx", []string{""}, false)
	Eval("set 600 hi nx xx", []string{"-ERR wrong syntax for 'set' command"}, false)
	// TODO: More 'SET' tests...

	Context("incr")
	Eval("incr 2000", []string{"1"}, false)
	Eval("incr 2000 5", []string{"6"}, false)
	Eval("incr 2000 f", []string{"-ERR increment is either not an integer or too large"}, false)
	Eval("incr 9000 9223372036854775807", []string{"9223372036854775807"}, false)
	Eval("incr 9000", []string{"-ERR value is either not an integer or too large"}, false)
	Eval("incr 9000 -1", []string{"-ERR inverse/non operations are discouraged"}, false)
	Eval("incr 600", []string{"-ERR value is either not an integer or too large"}, false)

	Context("decr")
	Eval("decr 20000", []string{"-1"}, false)
	Eval("decr 20000 5", []string{"-6"}, false)
	Eval("decr 20000 f", []string{"-ERR decrement is either not an integer or too large"}, false)
	Eval("decr 90000 9223372036854775807", []string{"-9223372036854775807"}, false)
	Eval("decr 90000", []string{"-9223372036854775808"}, false)
	Eval("decr 90000", []string{"-ERR value is either not an integer or too large"}, false)
	Eval("decr 90000 -1", []string{"-ERR inverse/non operations are discouraged"}, false)
	Eval("decr 600", []string{"-ERR value is either not an integer or too large"}, false)

	Context("append")
	Eval("append 80000 :)", []string{":)"}, false)
	Eval("append 80000 (:", []string{":)(:"}, false)

	Context("prepend")
	Eval("prepend 80000 \" \"", []string{" :)(:"}, false)
	Eval("prepend 80000 !", []string{"! :)(:"}, false)

	Context("len")
	Eval("len 80000", []string{"$6"}, false)
	Eval("len 80000 600 128", []string{"$6", "$2", "$-1"}, false)

	Context("rename")
	Eval("rename 880000 70000", []string{"-ERR no such key"}, false)
	Eval("rename 80000 70000", []string{"+OK"}, false)
	Assert("get 70000", []string{"! :)(:"}, false)
	Assert("get 80000", []string{""}, false)

	Context("copy")
	Eval("copy 80000 70000", []string{"-ERR no such key"}, false)
	Eval("copy 70000 80000", []string{"+OK"}, false)
	Assert("get 70000", []string{"! :)(:"}, false)
	Assert("get 80000", []string{"! :)(:"}, false)
	Eval("copy 70000 80000", []string{"-ERR destination key is not empty"}, false)
	Setup("set 70000 hello")
	Eval("copy 70000 80000 replace", []string{"+OK"}, false)
	Assert("get 70000", []string{"hello"}, false)
	Eval("copy 70000 80000 ???", []string{"-ERR unknown option"}, false)

	Context("getset")
	Eval("getset 70000 bye", []string{"hello"}, false)
	Assert("get 70000", []string{"bye"}, false)
	Eval("getset 70707 bye", []string{""}, false)
	Assert("get 70707", []string{"bye"}, false)

	Context("getdel")
	Eval("getdel 70707", []string{"bye"}, false)
	Assert("get 70707", []string{""}, false)
	Eval("getdel 70707", []string{""}, false)

	Context("del")
	Eval("del 70000 70707", []string{":1"}, false)
	Eval("del 80808", []string{":0"}, false)

	Context("exists")
	Eval("exists 70000 70707", []string{":0"}, false)
	Eval("exists 80000 70000 600", []string{":2"}, false)

	Context("info")
	Eval("info", []string{"release_os_arch:linux-amd64"}, true) // We use some settings that should hardly ever change
	Eval("info", []string{"memory_database_shards:50"}, true)
	Eval("info", []string{"memory_logical_databases:3"}, true)
	Eval("info", []string{"memory_active_database:0"}, true) // This we know for sure

	Context("echo")
	Eval("echo \"hello, world!\"", []string{"hello, world!"}, false)
	Eval("echo hi bye", []string{"-ERR wrong number of arguments for 'echo' command"}, false)
	Eval("echo", []string{"-ERR wrong number of arguments for 'echo' command"}, false)

	Context("flush")
	Eval("flush", []string{"+OK"}, false)
	Assert("info", []string{"keyspace_keys:0"}, true)

	// TODO: 'shutdown' command

	Context("quit") // Moved this down, hence a little out of order, see 'commands' package
	Eval("quit", []string{"+OK"}, false)

	/* Teardown */
	Read.Reset()
	if err := connection.Disconnect(Conn); err != nil {
		log.Fatalf("error: %s", err)
	}
}
