package commands

import (
	"net"
	"slices"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"lj.com/valhaj/internal/config"
	"lj.com/valhaj/internal/memory"
	"lj.com/valhaj/internal/statistics"
	"lj.com/valhaj/internal/writer"
)

// Command implements the behavior of the commands.
type Command struct {
	Arguments  []string
	Connection net.Conn
	Index      int
	Database   memory.ShardedCache
}

// Empty(): Checks if the command is empty, hence unnecessary.
func (cmd *Command) Empty() bool {
	return len(cmd.Arguments) == 0
}

// Execute(): Executes the command and writes the response. Returns false when the connection should be closed.
func (cmd *Command) Execute() (int, bool) {
	command := strings.ToUpper(cmd.Arguments[0])
	switch command {
	case "SELECT":
		return cmd.selectCommand()
	case "FLUSHALL":
		return cmd.flushallCommand()
	case "MOVE":
		return cmd.moveCommand()
	case "MGET":
		return cmd.mgetCommand()
	case "MSET":
		return cmd.msetCommand()
	case "GET":
		return cmd.getCommand()
	case "SET":
		return cmd.setCommand()
	case "INCR":
		return cmd.incrCommand()
	case "DECR":
		return cmd.decrCommand()
	case "APPEND":
		return cmd.appendCommand()
	case "PREPEND":
		return cmd.prependCommand()
	case "LEN":
		return cmd.lenCommand()
	case "RENAME":
		return cmd.renameCommand()
	case "COPY":
		return cmd.copyCommand()
	case "GETSET":
		return cmd.getsetCommand()
	case "GETDEL":
		return cmd.getdelCommand()
	case "DEL":
		return cmd.delCommand()
	case "EXISTS":
		return cmd.existsCommand()
	case "QUIT":
		return cmd.quitCommand()
	case "INFO":
		return cmd.infoCommand()
	case "ECHO":
		return cmd.echoCommand()
	case "FLUSH":
		return cmd.flushCommand()
	case "SHUTDOWN":
		return cmd.shutdownCommand()
	default:
		responses := []string{"!1\r\n", "-ERR unknown command '", command, "'\r\n"}
		if _, wErr := cmd.Connection.Write(writer.BuildResponse(responses)); wErr != nil {
			return cmd.Index, false
		}
	}
	return cmd.Index, true
}

/* multi-database commands */

// selectCommand(): Select the active logical database for the current session.
func (cmd *Command) selectCommand() (int, bool) {
	var wErr error
	var responses []string

	if len(cmd.Arguments) != 2 {
		responses = []string{"!1\r\n", "-ERR wrong number of arguments for '", cmd.Arguments[0], "' command\r\n"}
		if _, wErr = cmd.Connection.Write(writer.BuildResponse(responses)); wErr != nil {
			return cmd.Index, false
		}
		return cmd.Index, true
	}

	newIndex, err := strconv.Atoi(cmd.Arguments[1])
	if err != nil {
		responses = []string{"!1\r\n", "-ERR index value is not an integer\r\n"}
		if _, wErr = cmd.Connection.Write(writer.BuildResponse(responses)); wErr != nil {
			return cmd.Index, false
		}
		return cmd.Index, true
	}

	if newIndex < 0 || newIndex >= config.MemoryCacheContainerSize {
		responses = []string{"!1\r\n", "-ERR index value is out of bounds\r\n"}
		if _, wErr = cmd.Connection.Write(writer.BuildResponse(responses)); wErr != nil {
			return cmd.Index, false
		}
		return cmd.Index, true
	}

	responses = []string{"!1\r\n", "+OK\r\n"}
	if _, wErr = cmd.Connection.Write(writer.BuildResponse(responses)); wErr != nil {
		return newIndex, false
	}
	return newIndex, true
}

// flushallCommand(): Deletes all of the keys in every database. Requires elevated privileges.
func (cmd *Command) flushallCommand() (int, bool) {
	var wErr error
	var responses []string
	var wg sync.WaitGroup

	if len(cmd.Arguments) != 1 {
		responses = []string{"!1\r\n", "-ERR wrong number of arguments for '", cmd.Arguments[0], "' command\r\n"}
		if _, wErr = cmd.Connection.Write(writer.BuildResponse(responses)); wErr != nil {
			return cmd.Index, false
		}
		return cmd.Index, true
	}

	address := cmd.Connection.RemoteAddr()
	if !isAdmin(address) {
		responses = []string{"!1\r\n", "-ERR insufficient permissions\r\n"}
		if _, wErr = cmd.Connection.Write(writer.BuildResponse(responses)); wErr != nil {
			return cmd.Index, false
		}
		return cmd.Index, true
	}

	wg.Add(config.MemoryCacheContainerSize)
	for _, database := range memory.Container {
		go func(database memory.ShardedCache) {
			defer wg.Done()
			database.Clear()
		}(*database)
	}

	wg.Wait()

	responses = []string{"!1\r\n", "+OK\r\n"}
	if _, wErr = cmd.Connection.Write(writer.BuildResponse(responses)); wErr != nil {
		return cmd.Index, false
	}
	return cmd.Index, true
}

// moveCommand(): Move key from the currently selected database to the specified destination database.
func (cmd *Command) moveCommand() (int, bool) {
	var wErr error
	var responses []string

	if len(cmd.Arguments) != 3 {
		responses = []string{"!1\r\n", "-ERR wrong number of arguments for '", cmd.Arguments[0], "' command\r\n"}
		if _, wErr = cmd.Connection.Write(writer.BuildResponse(responses)); wErr != nil {
			return cmd.Index, false
		}
		return cmd.Index, true
	}

	newIndex, err := strconv.Atoi(cmd.Arguments[2])
	if err != nil {
		responses = []string{"!1\r\n", "-ERR index value is not an integer\r\n"}
		if _, wErr = cmd.Connection.Write(writer.BuildResponse(responses)); wErr != nil {
			return cmd.Index, false
		}
		return cmd.Index, true
	}

	if newIndex < 0 || newIndex >= config.MemoryCacheContainerSize {
		responses = []string{"!1\r\n", "-ERR index value is out of bounds\r\n"}
		if _, wErr = cmd.Connection.Write(writer.BuildResponse(responses)); wErr != nil {
			return cmd.Index, false
		}
		return cmd.Index, true
	}

	if newIndex == cmd.Index {
		responses = []string{"!1\r\n", "+OK\r\n"}
		if _, wErr = cmd.Connection.Write(writer.BuildResponse(responses)); wErr != nil {
			return cmd.Index, false
		}
		return cmd.Index, true
	}

	newDatabase := memory.Container[newIndex]

	// New key and db = new shard, hence the separate ops
	if value, ok := cmd.Database.Load(cmd.Arguments[1]); ok {
		if _, ok := newDatabase.LoadExistStore(cmd.Arguments[1], value, false, false); ok {
			responses = []string{"!1\r\n", "-ERR key already exists in destination database\r\n"}
			if _, wErr = cmd.Connection.Write(writer.BuildResponse(responses)); wErr != nil {
				return cmd.Index, false
			}
			return cmd.Index, true
		}
		cmd.Database.Delete(cmd.Arguments[1]) // And we'll only delete the key if it's movable
		responses = []string{"!1\r\n", "+OK\r\n"}
		if _, wErr = cmd.Connection.Write(writer.BuildResponse(responses)); wErr != nil {
			return cmd.Index, false
		}
	} else {
		responses = []string{"!1\r\n", "-ERR no such key\r\n"}
		if _, wErr = cmd.Connection.Write(writer.BuildResponse(responses)); wErr != nil {
			return cmd.Index, false
		}
	}
	return cmd.Index, true
}

/* single-database commands */

// mgetCommand(): Returns the values of all specified keys. If the key does not exist, an empty value is returned.
func (cmd *Command) mgetCommand() (int, bool) {
	var wErr error

	clen := len(cmd.Arguments[1:])
	if clen < 1 {
		responses := []string{"!1\r\n", "-ERR wrong number of arguments for '", cmd.Arguments[0], "' command\r\n"}
		if _, wErr = cmd.Connection.Write(writer.BuildResponse(responses)); wErr != nil {
			return cmd.Index, false
		}
		return cmd.Index, true
	}

	maxSize := clen*2 + 3 // INFO: Maximum possible size (clen * 2 ['value' + '\r\n', or just '\r\n'] + 3 protocol)
	var responses = make([]string, 0, maxSize)
	responses = append(responses, "!", strconv.Itoa(clen), "\r\n")
	for i := 1; i <= clen; i++ {
		value, ok := cmd.Database.Load(cmd.Arguments[i])
		if ok {
			responses = append(responses, value, "\r\n")
		} else {
			responses = append(responses, "\r\n")
		}
	}

	if _, wErr = cmd.Connection.Write(writer.BuildResponse(responses)); wErr != nil {
		return cmd.Index, false
	}
	return cmd.Index, true
}

// msetCommand(): Sets the given keys to their respective values, replacing existing values.
func (cmd *Command) msetCommand() (int, bool) {
	var wErr error
	var responses []string

	clen := len(cmd.Arguments[1:])
	if clen%2 != 0 || clen == 0 {
		responses = []string{"!1\r\n", "-ERR wrong number of arguments for '", cmd.Arguments[0], "' command\r\n"}
		if _, wErr = cmd.Connection.Write(writer.BuildResponse(responses)); wErr != nil {
			return cmd.Index, false
		}
		return cmd.Index, true
	}

	for i := 2; i <= clen; i += 2 {
		cmd.Database.Store(cmd.Arguments[i-1], cmd.Arguments[i])
	}

	responses = []string{"!1\r\n", "+OK\r\n"}
	if _, wErr = cmd.Connection.Write(writer.BuildResponse(responses)); wErr != nil {
		return cmd.Index, false
	}
	return cmd.Index, true
}

// getCommand(): Retrieves the value of a key if it exists.
func (cmd *Command) getCommand() (int, bool) {
	var wErr error
	var responses []string

	if len(cmd.Arguments) != 2 {
		responses = []string{"!1\r\n", "-ERR wrong number of arguments for '", cmd.Arguments[0], "' command\r\n"}
		if _, wErr = cmd.Connection.Write(writer.BuildResponse(responses)); wErr != nil {
			return cmd.Index, false
		}
		return cmd.Index, true
	}

	value, ok := cmd.Database.Load(cmd.Arguments[1])
	if ok {
		responses = []string{"!1\r\n", value, "\r\n"}
		_, wErr = cmd.Connection.Write(writer.BuildResponse(responses))
	} else {
		responses = []string{"!1\r\n", "\r\n"}
		_, wErr = cmd.Connection.Write(writer.BuildResponse(responses))
	}
	if wErr != nil {
		return cmd.Index, false
	}
	return cmd.Index, true
}

// setCommand(): Stores a key value pair. Optionally sets expiration on the key.
func (cmd *Command) setCommand() (int, bool) {
	syntaxError, checkExist, checkExpire := false, false, false
	var optExist, optExpire, durExpire string
	var wErr error
	var responses []string

	clen := len(cmd.Arguments)
	if clen < 3 || clen > 6 {
		responses = []string{"!1\r\n", "-ERR wrong number of arguments for '", cmd.Arguments[0], "' command\r\n"}
		if _, wErr = cmd.Connection.Write(writer.BuildResponse(responses)); wErr != nil {
			return cmd.Index, false
		}
		return cmd.Index, true
	}

	// Parse
	if clen > 3 {
		for idx := 3; idx < clen; idx++ {
			option := strings.ToUpper(cmd.Arguments[idx])
			if slices.Contains([]string{"NX", "XX"}, option) && !checkExist {
				optExist = option
				checkExist = true
			} else if slices.Contains([]string{"PX", "EX"}, option) && !checkExpire {
				optExpire = option
				idx += 1
				if idx >= clen {
					syntaxError = true
				} else {
					durExpire = cmd.Arguments[idx]
					checkExpire = true
				}
			} else {
				syntaxError = true
			}
		}
	}

	// Run
	var exists bool
	if syntaxError {
		responses = []string{"!1\r\n", "-ERR wrong syntax for '", cmd.Arguments[0], "' command\r\n"}
		if _, wErr = cmd.Connection.Write(writer.BuildResponse(responses)); wErr != nil {
			return cmd.Index, false
		}
	} else {
		if checkExist {
			if optExist == "XX" {
				exists = true
			} else { // NX
				exists = false
			}

			if _, ok := cmd.Database.LoadExistStore(cmd.Arguments[1], cmd.Arguments[2], exists, false); ok == exists {
				responses = []string{"!1\r\n", "+OK\r\n"}
				_, wErr = cmd.Connection.Write(writer.BuildResponse(responses))
			} else {
				responses = []string{"!1\r\n", "\r\n"}
				_, wErr = cmd.Connection.Write(writer.BuildResponse(responses))
			}
			if wErr != nil {
				return cmd.Index, false
			}
		} else {
			cmd.Database.Store(cmd.Arguments[1], cmd.Arguments[2])
			responses = []string{"!1\r\n", "+OK\r\n"}
			if _, wErr = cmd.Connection.Write(writer.BuildResponse(responses)); wErr != nil {
				return cmd.Index, false
			}
		}

		if checkExpire {
			setExpiration(cmd.Arguments[1], optExpire, durExpire, cmd.Database)
		}
	}

	return cmd.Index, true
}

// incrCommand(): Increments the integer value stored at key by the increment, creating it prior if it doesn't exist.
func (cmd *Command) incrCommand() (int, bool) {
	var wErr error
	var responses []string

	clen := len(cmd.Arguments)
	if clen < 2 || clen > 3 {
		responses = []string{"!1\r\n", "-ERR wrong number of arguments for '", cmd.Arguments[0], "' command\r\n"}
		if _, wErr = cmd.Connection.Write(writer.BuildResponse(responses)); wErr != nil {
			return cmd.Index, false
		}
		return cmd.Index, true
	}

	var err error
	increment := 1
	if clen == 3 {
		increment, err = strconv.Atoi(cmd.Arguments[2])
		if err != nil {
			responses = []string{"!1\r\n", "-ERR increment is either not an integer or too large\r\n"}
			if _, wErr = cmd.Connection.Write(writer.BuildResponse(responses)); wErr != nil {
				return cmd.Index, false
			}
			return cmd.Index, true
		}
		if increment < 1 {
			responses = []string{"!1\r\n", "-ERR inverse/non operations are discouraged\r\n"}
			if _, wErr = cmd.Connection.Write(writer.BuildResponse(responses)); wErr != nil {
				return cmd.Index, false
			}
			return cmd.Index, true
		}
	}

	value, status := cmd.Database.LoadModifyStore(
		cmd.Arguments[1],
		func(v string) (string, bool) {
			n, err := strconv.Atoi(v)
			if err != nil {
				return v, false
			}
			positive := n > 0
			n += increment
			if positive && n < 0 {
				return v, false
			}
			return strconv.Itoa(n), true
		},
		"0",
	)

	if !status {
		responses = []string{"!1\r\n", "-ERR value is either not an integer or too large\r\n"}
		_, wErr = cmd.Connection.Write(writer.BuildResponse(responses))
	} else {
		responses = []string{"!1\r\n", value, "\r\n"}
		_, wErr = cmd.Connection.Write(writer.BuildResponse(responses))
	}
	if wErr != nil {
		return cmd.Index, false
	}
	return cmd.Index, true
}

// decrCommand(): Decrements the integer value stored at key by the decrement, creating it prior if it doesn't exist.
func (cmd *Command) decrCommand() (int, bool) {
	var wErr error
	var responses []string

	clen := len(cmd.Arguments)
	if clen < 2 || clen > 3 {
		responses = []string{"!1\r\n", "-ERR wrong number of arguments for '", cmd.Arguments[0], "' command\r\n"}
		if _, wErr = cmd.Connection.Write(writer.BuildResponse(responses)); wErr != nil {
			return cmd.Index, false
		}
		return cmd.Index, true
	}

	var err error
	decrement := 1
	if clen == 3 {
		decrement, err = strconv.Atoi(cmd.Arguments[2])
		if err != nil {
			responses = []string{"!1\r\n", "-ERR decrement is either not an integer or too large\r\n"}
			if _, wErr = cmd.Connection.Write(writer.BuildResponse(responses)); wErr != nil {
				return cmd.Index, false
			}
			return cmd.Index, true
		}
		if decrement < 1 {
			responses = []string{"!1\r\n", "-ERR inverse/non operations are discouraged\r\n"}
			if _, wErr = cmd.Connection.Write(writer.BuildResponse(responses)); wErr != nil {
				return cmd.Index, false
			}
			return cmd.Index, true
		}
	}

	value, status := cmd.Database.LoadModifyStore(
		cmd.Arguments[1],
		func(v string) (string, bool) {
			n, err := strconv.Atoi(v)
			if err != nil {
				return v, false
			}
			negative := n < 0
			n -= decrement
			if negative && n > 0 {
				return v, false
			}
			return strconv.Itoa(n), true
		},
		"0",
	)

	if !status {
		responses = []string{"!1\r\n", "-ERR value is either not an integer or too large\r\n"}
		_, wErr = cmd.Connection.Write(writer.BuildResponse(responses))
	} else {
		responses = []string{"!1\r\n", value, "\r\n"}
		_, wErr = cmd.Connection.Write(writer.BuildResponse(responses))
	}
	if wErr != nil {
		return cmd.Index, false
	}
	return cmd.Index, true
}

// appendCommand(): Appends to the value stored at key, creating it prior if it doesn't exist.
func (cmd *Command) appendCommand() (int, bool) {
	var wErr error
	var responses []string

	if len(cmd.Arguments) != 3 {
		responses = []string{"!1\r\n", "-ERR wrong number of arguments for '", cmd.Arguments[0], "' command\r\n"}
		if _, wErr = cmd.Connection.Write(writer.BuildResponse(responses)); wErr != nil {
			return cmd.Index, false
		}
		return cmd.Index, true
	}

	value, _ := cmd.Database.LoadModifyStore(
		cmd.Arguments[1],
		func(v string) (string, bool) {
			return strings.Join([]string{v, cmd.Arguments[2]}, ""), true
		},
		"",
	)

	responses = []string{"!1\r\n", value, "\r\n"}
	if _, wErr = cmd.Connection.Write(writer.BuildResponse(responses)); wErr != nil {
		return cmd.Index, false
	}
	return cmd.Index, true
}

// prependCommand(): Prepends to the value stored at key, creating it prior if it doesn't exist.
func (cmd *Command) prependCommand() (int, bool) {
	var wErr error
	var responses []string

	if len(cmd.Arguments) != 3 {
		responses = []string{"!1\r\n", "-ERR wrong number of arguments for '", cmd.Arguments[0], "' command\r\n"}
		if _, wErr = cmd.Connection.Write(writer.BuildResponse(responses)); wErr != nil {
			return cmd.Index, false
		}
		return cmd.Index, true
	}

	value, _ := cmd.Database.LoadModifyStore(
		cmd.Arguments[1],
		func(v string) (string, bool) {
			return strings.Join([]string{cmd.Arguments[2], v}, ""), true
		},
		"",
	)

	responses = []string{"!1\r\n", value, "\r\n"}
	if _, wErr = cmd.Connection.Write(writer.BuildResponse(responses)); wErr != nil {
		return cmd.Index, false
	}
	return cmd.Index, true
}

// lenCommand(): Returns the value length of all the specified keys.
func (cmd *Command) lenCommand() (int, bool) {
	var wErr error

	clen := len(cmd.Arguments[1:])
	if clen < 1 {
		responses := []string{"!1\r\n", "-ERR wrong number of arguments for '", cmd.Arguments[0], "' command\r\n"}
		if _, wErr = cmd.Connection.Write(writer.BuildResponse(responses)); wErr != nil {
			return cmd.Index, false
		}
		return cmd.Index, true
	}

	maxSize := clen*3 + 3
	var responses = make([]string, 0, maxSize)
	responses = append(responses, "!", strconv.Itoa(clen), "\r\n")
	for _, k := range cmd.Arguments[1:] {
		if v, ok := cmd.Database.Load(k); ok {
			responses = append(responses, "$", strconv.Itoa(len(v)), "\r\n")
		} else {
			responses = append(responses, "$-1\r\n")
		}
	}

	if _, wErr = cmd.Connection.Write(writer.BuildResponse(responses)); wErr != nil {
		return cmd.Index, false
	}
	return cmd.Index, true
}

// renameCommand(): Renames key to newkey, returning an error if key doesn't exist and overwriting newkey if it exists.
func (cmd *Command) renameCommand() (int, bool) {
	var wErr error
	var responses []string

	if len(cmd.Arguments) != 3 {
		responses = []string{"!1\r\n", "-ERR wrong number of arguments for '", cmd.Arguments[0], "' command\r\n"}
		if _, wErr = cmd.Connection.Write(writer.BuildResponse(responses)); wErr != nil {
			return cmd.Index, false
		}
		return cmd.Index, true
	}

	// New key = new shard, hence the separate load and store ops
	if value, ok := cmd.Database.LoadAndDelete(cmd.Arguments[1]); ok {
		cmd.Database.Store(cmd.Arguments[2], value)
		responses = []string{"!1\r\n", "+OK\r\n"}
		_, wErr = cmd.Connection.Write(writer.BuildResponse(responses))
	} else {
		responses = []string{"!1\r\n", "-ERR no such key\r\n"}
		_, wErr = cmd.Connection.Write(writer.BuildResponse(responses))
	}
	if wErr != nil {
		return cmd.Index, false
	}
	return cmd.Index, true
}

// copyCommand(): Copies the value stored at the source key to the destination key, replacing the existing value if desired.
func (cmd *Command) copyCommand() (int, bool) {
	var wErr error
	var responses []string

	clen := len(cmd.Arguments)
	if clen < 3 || clen > 4 {
		responses = []string{"!1\r\n", "-ERR wrong number of arguments for '", cmd.Arguments[0], "' command\r\n"}
		if _, wErr = cmd.Connection.Write(writer.BuildResponse(responses)); wErr != nil {
			return cmd.Index, false
		}
		return cmd.Index, true
	}

	exists := false
	overwrite := false
	replace := "REPLACE"
	if clen == 4 {
		if strings.ToUpper(cmd.Arguments[3]) != replace {
			responses = []string{"!1\r\n", "-ERR unknown option\r\n"}
			if _, wErr = cmd.Connection.Write(writer.BuildResponse(responses)); wErr != nil {
				return cmd.Index, false
			}
			return cmd.Index, true
		}
		overwrite = true
	}

	// New key = new shard, hence the separate load and store ops
	if value, ok := cmd.Database.Load(cmd.Arguments[1]); ok {
		_, ok = cmd.Database.LoadExistStore(cmd.Arguments[2], value, exists, overwrite)
		if ok == exists || overwrite {
			responses = []string{"!1\r\n", "+OK\r\n"}
			_, wErr = cmd.Connection.Write(writer.BuildResponse(responses))
		} else {
			responses = []string{"!1\r\n", "-ERR destination key is not empty\r\n"}
			_, wErr = cmd.Connection.Write(writer.BuildResponse(responses))
		}
	} else {
		responses = []string{"!1\r\n", "-ERR no such key\r\n"}
		_, wErr = cmd.Connection.Write(writer.BuildResponse(responses))
	}
	if wErr != nil {
		return cmd.Index, false
	}
	return cmd.Index, true
}

// getsetCommand(): Atomically sets the key to the new value and returns the old value.
func (cmd *Command) getsetCommand() (int, bool) {
	var wErr error
	var responses []string

	if len(cmd.Arguments) != 3 {
		responses = []string{"!1\r\n", "-ERR wrong number of arguments for '", cmd.Arguments[0], "' command\r\n"}
		if _, wErr = cmd.Connection.Write(writer.BuildResponse(responses)); wErr != nil {
			return cmd.Index, false
		}
		return cmd.Index, true
	}

	value, _ := cmd.Database.LoadExistStore(cmd.Arguments[1], cmd.Arguments[2], true, true)
	responses = []string{"!1\r\n", value, "\r\n"}
	if _, wErr = cmd.Connection.Write(writer.BuildResponse(responses)); wErr != nil {
		return cmd.Index, false
	}
	return cmd.Index, true
}

// getdelCommand(): Retrieves the value of a key if it exists and deletes the key.
func (cmd *Command) getdelCommand() (int, bool) {
	var wErr error
	var responses []string

	if len(cmd.Arguments) != 2 {
		responses = []string{"!1\r\n", "-ERR wrong number of arguments for '", cmd.Arguments[0], "' command\r\n"}
		if _, wErr = cmd.Connection.Write(writer.BuildResponse(responses)); wErr != nil {
			return cmd.Index, false
		}
		return cmd.Index, true
	}

	value, ok := cmd.Database.LoadAndDelete(cmd.Arguments[1])
	if ok {
		responses = []string{"!1\r\n", value, "\r\n"}
		_, wErr = cmd.Connection.Write(writer.BuildResponse(responses))
	} else {
		responses = []string{"!1\r\n", "\r\n"}
		_, wErr = cmd.Connection.Write(writer.BuildResponse(responses))
	}
	if wErr != nil {
		return cmd.Index, false
	}
	return cmd.Index, true
}

// delCommand(): Removes the specified keys. A key is ignored if it does not exist.
func (cmd *Command) delCommand() (int, bool) {
	var wErr error
	var responses []string

	if len(cmd.Arguments) < 2 {
		responses = []string{"!1\r\n", "-ERR wrong number of arguments for '", cmd.Arguments[0], "' command\r\n"}
		if _, wErr = cmd.Connection.Write(writer.BuildResponse(responses)); wErr != nil {
			return cmd.Index, false
		}
		return cmd.Index, true
	}

	count := 0
	for _, k := range cmd.Arguments[1:] {
		if _, ok := cmd.Database.LoadAndDelete(k); ok {
			count++
		}
	}

	responses = []string{"!1\r\n", ":", strconv.Itoa(count), "\r\n"}
	if _, wErr = cmd.Connection.Write(writer.BuildResponse(responses)); wErr != nil {
		return cmd.Index, false
	}
	return cmd.Index, true
}

// existsCommand(): Checks if a key exists.
func (cmd *Command) existsCommand() (int, bool) {
	var wErr error
	var responses []string

	if len(cmd.Arguments) < 2 {
		responses = []string{"!1\r\n", "-ERR wrong number of arguments for '", cmd.Arguments[0], "' command\r\n"}
		if _, wErr = cmd.Connection.Write(writer.BuildResponse(responses)); wErr != nil {
			return cmd.Index, false
		}
		return cmd.Index, true
	}

	count := 0
	for _, k := range cmd.Arguments[1:] {
		if _, ok := cmd.Database.Load(k); ok {
			count++
		}
	}

	responses = []string{"!1\r\n", ":", strconv.Itoa(count), "\r\n"}
	if _, wErr = cmd.Connection.Write(writer.BuildResponse(responses)); wErr != nil {
		return cmd.Index, false
	}
	return cmd.Index, true
}

// quitCommand(): Instructs the server to terminate the connection.
func (cmd *Command) quitCommand() (int, bool) {
	var wErr error
	var responses []string

	if len(cmd.Arguments) != 1 {
		responses = []string{"!1\r\n", "-ERR wrong number of arguments for '", cmd.Arguments[0], "' command\r\n"}
		if _, wErr = cmd.Connection.Write(writer.BuildResponse(responses)); wErr != nil {
			return cmd.Index, false
		}
		return cmd.Index, true
	}
	responses = []string{"!1\r\n", "+OK\r\n"}
	_, _ = cmd.Connection.Write(writer.BuildResponse(responses))
	return cmd.Index, false
}

// infoCommand(): Returns information and statistics about the server in a simple format.
func (cmd *Command) infoCommand() (int, bool) {
	var wErr error

	if len(cmd.Arguments) != 1 {
		responses := []string{"!1\r\n", "-ERR wrong number of arguments for '", cmd.Arguments[0], "' command\r\n"}
		if _, wErr = cmd.Connection.Write(writer.BuildResponse(responses)); wErr != nil {
			return cmd.Index, false
		}
		return cmd.Index, true
	}

	totalKeys, _ := cmd.Database.Count()
	stats := statistics.GetStats(cmd.Index, totalKeys)
	statCount := len(stats)

	maxSize := statCount*2 + 3
	var responses = make([]string, 0, maxSize)
	responses = append(responses, "!", strconv.Itoa(statCount), "\r\n")
	for _, stat := range stats {
		responses = append(responses, stat, "\r\n")
	}

	if _, wErr = cmd.Connection.Write(writer.BuildResponse(responses)); wErr != nil {
		return cmd.Index, false
	}
	return cmd.Index, true
}

// echoCommand(): Returns the message sent by the client. May serve benchmarking purposes.
func (cmd *Command) echoCommand() (int, bool) {
	var wErr error
	var responses []string

	if len(cmd.Arguments) != 2 {
		responses = []string{"!1\r\n", "-ERR wrong number of arguments for '", cmd.Arguments[0], "' command\r\n"}
		if _, wErr = cmd.Connection.Write(writer.BuildResponse(responses)); wErr != nil {
			return cmd.Index, false
		}
		return cmd.Index, true
	}

	responses = []string{"!1\r\n", cmd.Arguments[1], "\r\n"}
	if _, wErr = cmd.Connection.Write(writer.BuildResponse(responses)); wErr != nil {
		return cmd.Index, false
	}
	return cmd.Index, true
}

// flushCommand(): Deletes all of the keys in the current database. Requires elevated privileges.
func (cmd *Command) flushCommand() (int, bool) {
	var wErr error
	var responses []string

	if len(cmd.Arguments) != 1 {
		responses = []string{"!1\r\n", "-ERR wrong number of arguments for '", cmd.Arguments[0], "' command\r\n"}
		if _, wErr = cmd.Connection.Write(writer.BuildResponse(responses)); wErr != nil {
			return cmd.Index, false
		}
		return cmd.Index, true
	}

	address := cmd.Connection.RemoteAddr()
	if isAdmin(address) {
		cmd.Database.Clear()
		responses = []string{"!1\r\n", "+OK\r\n"}
		_, wErr = cmd.Connection.Write(writer.BuildResponse(responses))
	} else {
		responses = []string{"!1\r\n", "-ERR insufficient permissions\r\n"}
		_, wErr = cmd.Connection.Write(writer.BuildResponse(responses))
	}
	if wErr != nil {
		return cmd.Index, false
	}
	return cmd.Index, true
}

// shutdownCommand(): Used to externally trigger a graceful shutdown. Requires elevated privileges.
func (cmd *Command) shutdownCommand() (int, bool) {
	var wErr error
	var responses []string

	if len(cmd.Arguments) != 1 {
		responses = []string{"!1\r\n", "-ERR wrong number of arguments for '", cmd.Arguments[0], "' command\r\n"}
		if _, wErr = cmd.Connection.Write(writer.BuildResponse(responses)); wErr != nil {
			return cmd.Index, false
		}
		return cmd.Index, true
	}

	address := cmd.Connection.RemoteAddr()
	if isAdmin(address) {
		syscall.Kill(statistics.ProcessId, syscall.SIGINT)
		responses = []string{"!1\r\n", "+OK\r\n"}
		_, _ = cmd.Connection.Write(writer.BuildResponse(responses))
		return cmd.Index, false
	}
	responses = []string{"!1\r\n", "-ERR insufficient permissions\r\n"}
	if _, wErr = cmd.Connection.Write(writer.BuildResponse(responses)); wErr != nil {
		return cmd.Index, false
	}
	return cmd.Index, true
}

/* extras */

// setExpiration(): Handles expiration when passed as part of the 'SET' command.
func setExpiration(key string, option string, ttl string, database memory.ShardedCache) {
	value, _ := strconv.Atoi(ttl) // Returns '0' on error
	if value == 0 {               // No need to start a goroutine for that
		return
	}

	var duration time.Duration
	if option == "PX" {
		duration = time.Millisecond * time.Duration(value)
	} else { // EX
		duration = time.Second * time.Duration(value)
	}

	go func() {
		time.Sleep(duration)
		database.Delete(key)
	}()
}

// isAdmin(): Checks whether or not the current client is connected locally, thus having administrative permissions.
func isAdmin(address net.Addr) bool {
	localIPv4 := "127.0.0.1"
	localIPv6 := "[::1]"
	networkUNIX := "unix" // INFO: There's "unix", "unixgram" and "unixpacket"

	addressString := address.String()
	addressNetwork := address.Network()
	if addressNetwork == networkUNIX {
		return true
	} else if strings.Contains(addressString, localIPv4) || strings.Contains(addressString, localIPv6) {
		return true
	}
	return false
}
