package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"lj.com/valhaj-proxy/internal/config"
	"lj.com/valhaj-proxy/internal/server"
)

func main() {
	fmt.Printf(
		"Starting %s, version %s. Copyright (C) %s %s.\n",
		config.ReleaseTitle,
		config.ReleaseVersion,
		config.ReleaseYear,
		config.ReleaseAuthor,
	)

	// Main server handling
	s := server.NewServer(
		config.ServerProxyNetwork,
		config.ServerProxyAddress,
		config.ServerCAFile,
		config.ServerCertFile,
		config.ServerKeyFile,
	)
	s.WG.Add(1)
	go s.Serve()

	quitChannel := make(chan os.Signal, 1)
	signal.Notify(quitChannel, syscall.SIGINT, syscall.SIGTERM)
	<-quitChannel
	fmt.Printf("\n")
	s.Quit()
}
