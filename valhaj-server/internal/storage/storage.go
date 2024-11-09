package storage

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"

	"lj.com/valhaj/internal/config"
	"lj.com/valhaj/internal/memory"
)

// CreateLabels(): Generates filenames for each database state backup.
func CreateLabels() []string {
	var containerSize = config.MemoryCacheContainerSize
	var filename string
	var labels = make([]string, 0, containerSize)

	for i := 0; i < containerSize; i++ {
		filename = strings.Join([]string{config.StorageBasename, strconv.Itoa(i), config.StorageExtension}, "")
		labels = append(labels, filename)
	}

	return labels
}

// DiskWrite(): Persists the database state to disk, if it's not empty.
func DiskWrite(filename string, database memory.ShardedCache, index int) error {
	items, count := database.Range()
	if len(items) == 0 {
		return fmt.Errorf("no data to persist to disk")
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("error creating snapshot file (%w)", err)
	}
	defer file.Close()

	for _, item := range items {
		_, err := fmt.Fprintf(file, "%v\n", item)
		if err != nil {
			return fmt.Errorf("error writing snapshot file (%w)", err)
		}
	}

	log.Printf("Saved database snapshot id=%d containing %d key(s)\n", index, count)
	return nil
}

// DiskRead(): Attempts to restore an old database state, if it exists.
func DiskRead(filename string, database memory.ShardedCache, index int) error {
	file, err := os.ReadFile(filename)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("no snapshot to restore")
		}
		return fmt.Errorf("error reading snapshot from disk (%w)", err)
	}

	fileContent := string(file)
	fileRows := strings.Split(fileContent, "\n")
	rowCount := len(fileRows)
	if rowCount > 0 {
		fileRows = fileRows[:rowCount-1]
		rowCount -= 1
	}
	if rowCount%2 != 0 {
		return fmt.Errorf("error loading incomplete snapshot")
	}

	pair := 0
	var kv []string
	for _, row := range fileRows {
		kv = append(kv, row)
		pair += 1
		if pair == 2 {
			database.Store(kv[0], kv[1])
			pair = 0
			kv = nil
		}
	}

	count := rowCount / 2
	log.Printf("Restored database snapshot id=%d containing %d key(s)\n", index, count)
	return nil
}

// SaveState(): Persists the state of all databases to disk.
func SaveState() {
	var wg sync.WaitGroup

	fileNames := CreateLabels()
	wg.Add(config.MemoryCacheContainerSize)
	for index, fileName := range fileNames {
		database := *memory.Container[index]
		go func(index int, fileName string) {
			defer wg.Done()
			if err := DiskWrite(fileName, database, index); err != nil {
				log.Printf("Skipped saving database snapshot id=%d: %s\n", index, err)
			}
		}(index, fileName)
	}

	wg.Wait()
}

// RestoreState(): Restores the previous state of all databases.
func RestoreState() {
	var wg sync.WaitGroup

	fileNames := CreateLabels()
	wg.Add(config.MemoryCacheContainerSize)
	for index, fileName := range fileNames {
		database := *memory.Container[index]
		go func(index int, fileName string) {
			defer wg.Done()
			if err := DiskRead(fileName, database, index); err != nil {
				log.Printf("Skipped restoring database snapshot id=%d: %s\n", index, err)
			}
		}(index, fileName)
	}

	wg.Wait()
}
