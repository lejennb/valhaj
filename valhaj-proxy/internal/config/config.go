package config

const (
	/* cmd/proxy */
	ReleaseTitle   = "valhaj-proxy"
	ReleaseVersion = "1.0.0-dev"
	ReleaseYear    = "2024"
	ReleaseAuthor  = "lejennb"
	/* internal/server */
	ServerProxyNetwork          = "tcp"
	ServerProxyAddress          = "0.0.0.0:6380"
	ServerDatabaseNetwork       = "unix"
	ServerDatabaseAddress       = "/tmp/valhaj.sock"
	ServerCAFile                = "./ca-cert.pem"
	ServerCertFile              = "./server-cert.pem"
	ServerKeyFile               = "./server-key.pem"
	ServerGracefulShutdownDelay = 1000
)
