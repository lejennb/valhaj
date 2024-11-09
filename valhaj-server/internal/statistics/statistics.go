package statistics

import (
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"lj.com/valhaj/internal/config"
)

var (
	StartTime time.Time
	ProcessId int
)

// InitStats(): Initializes static metrics to serve as the starting point for future offset operations.
func InitStats() (time.Time, int) {
	return time.Now(), syscall.Getpid()
}

// GetStats(): Returns an updated list of dynamic and static metrics for the current session.
func GetStats(index, totalKeys int) []string {
	return []string{
		strings.Join([]string{"server_pid:", strconv.Itoa(ProcessId)}, ""),
		strings.Join([]string{"server_uptime:", time.Since(StartTime).Round(time.Second).String()}, ""),
		strings.Join([]string{"server_version:", config.ReleaseVersion}, ""),
		strings.Join([]string{"server_network:", config.ServerInetNetwork}, ""),
		strings.Join([]string{"system_logical_cpus:", strconv.Itoa(runtime.NumCPU())}, ""),
		strings.Join([]string{"runtime_current_threads:", strconv.Itoa(runtime.NumGoroutine())}, ""),
		strings.Join([]string{"release_os_arch:", runtime.GOOS, "-", runtime.GOARCH}, ""),
		strings.Join([]string{"release_go_version:", runtime.Version()}, ""),
		strings.Join([]string{"keyspace_keys:", strconv.Itoa(totalKeys)}, ""),
		strings.Join([]string{"memory_database_shards:", strconv.Itoa(config.MemoryCacheShardCount)}, ""),
		strings.Join([]string{"memory_logical_databases:", strconv.Itoa(config.MemoryCacheContainerSize)}, ""),
		strings.Join([]string{"memory_active_database:", strconv.Itoa(index)}, ""),
	}
}
