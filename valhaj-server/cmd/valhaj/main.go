package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"lj.com/valhaj/internal/config"
	"lj.com/valhaj/internal/memory"
	"lj.com/valhaj/internal/server"
	"lj.com/valhaj/internal/statistics"
	"lj.com/valhaj/internal/storage"
)

func main() {
	fmt.Printf(
		"Welcome to %s, version %s. Copyright (C) %s %s.\n",
		config.ReleaseTitle,
		config.ReleaseVersion,
		config.ReleaseYear,
		config.ReleaseAuthor,
	)

	// Initialize statistics
	statistics.StartTime, statistics.ProcessId = statistics.InitStats()

	// Create caches
	memory.Container = memory.NewCacheContainer(config.MemoryCacheContainerSize, config.MemoryCacheShardCount)

	// Restore snapshots
	storage.RestoreState()

	// Main server handling
	s := server.NewServer(config.ServerInetNetwork, config.ServerInetAddress)
	s.WG.Add(1)
	go s.Serve()

	quitChannel := make(chan os.Signal, 1)
	signal.Notify(quitChannel, syscall.SIGINT, syscall.SIGTERM)
	<-quitChannel
	fmt.Printf("\n")
	s.Quit()

	// Write snapshots to disk
	storage.SaveState()

	log.Println("Bye")
}
