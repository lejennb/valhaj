package config

const (
	/* cmd/valhaj */
	ReleaseTitle   = "valhaj"
	ReleaseVersion = "1.0.31"
	ReleaseYear    = "2024"
	ReleaseAuthor  = "lejennb"
	/* internal/server */
	ServerUnixNetwork           = "unix"
	ServerUnixAddress           = "/tmp/valhaj.sock"
	ServerInetNetwork           = "tcp"
	ServerInetAddress           = "0.0.0.0:6380"
	ServerGracefulShutdownDelay = 1000
	/* internal/storage */
	StorageBasename  = "data"
	StorageExtension = ".vdb"
	/* internal/memory */
	MemoryCacheContainerSize = 3
	MemoryCacheShardCount    = 50
)
