# Valhaj Documentation

### Protocol
```
Valhaj follows its own cutom protocol in order to simplify and streamline client-server communication.
Generally speaking, Valhaj has the following response types:
```

| Type | Description |
| ---- | ----------- |
| `!n` | Provides the total number of additional responses, this should *always* be the first response - starting with its implementation in 1.0.16-dev. Example: `!7` indicates that the processed command will return another 7 responses. |
| `+OK` | Indicates successful command execution if no data is transmitted otherwise. |
| `-ERR ...` | Transmits the error that was encountered during server-side processing. Example: `-ERR wrong number of arguments for 'GET'`. Instead of writing long strings, there may be a range of predefined error codes. They can be cached client-side and displayed on demand, based on the error code. |
| `:n` | Indicates a count of something, usually results of a query. Example: `:0` as the static response of the CONFIG command (1.0.15-dev). |
| `$n` | Provides the length of a keys value or alternatively `$-1` if the key does not exist. Example: `$5` incidates that the value has a length of 5 characters. |

```
Every other response is actual data returned by the server.

Response types aim to aid humans as well as machine clients in understanding the result of a command, while also providing other useful information.
For example, the response count vastly simplifies client handling of the TCP communication.

The latter is of utmost importance in order to actually parse all reponses into an array as part of the client-side handling, one has to know how many responses will be returned.
It is not possible to read from a TCP connection until nothing "is sent anymore", as eventually there will be reads where 1. no data is ever transmitted or 2. there is data incoming in reponse to totally different commands.
On top of that, waiting for "io.EOF" is not an option, because EOF marks the closure of the underlying TCP connection, not the temporary absence of data being transmitted.
```

### Commands
```
```
