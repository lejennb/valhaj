# valhaj
An in-memory database that persists on disk. Written in Go, without any external dependencies.

### Building
* To build from source, simply run `make build`. For this to work, you need to have a suitable `Go` release installed on your system.
* Alternatively, you may also download a precompiled binary release.

### Network
* If you wish to use UNIX socket connections (local) instead of TCP connections, change `ServerNetwork` to `"unix"` and `ServerAddress` to a suitable path, like `"/tmp/valhaj.sock"`.
* You can then either connect to it by using the `go-valhaj` library or `netcat` (netcat-openbsd): `nc -C -U /tmp/valhaj.sock`.
* When using `ServerNetwork` = `"tcp"`, you may also use `go-valhaj` or `telnet`, e.g.: `telnet localhost 6380`.
