package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
)

type AOF struct {
	file *os.File
	mu   sync.Mutex
}

func NewAOF(path string) (*AOF, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}
	return &AOF{file: file}, nil
}

func (aof *AOF) Close() error {
	aof.mu.Lock()
	defer aof.mu.Unlock()
	return aof.file.Close()
}

// Append logs a SET command (simplified format for this project)
func (aof *AOF) AppendSet(key, value string, ttlSeconds int) error {
	aof.mu.Lock()
	defer aof.mu.Unlock()

	line := fmt.Sprintf("SET %s %s %d\n", key, value, ttlSeconds)
	_, err := aof.file.WriteString(line)
	return err
}

// Replay reads the AOF file and repopulates the store on startup
func (aof *AOF) Replay(store *Store) error {
	aof.mu.Lock()
	defer aof.mu.Unlock()

	// Seek to beginning
	_, err := aof.file.Seek(0, 0)
	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(aof.file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, " ")
		if len(parts) >= 4 && parts[0] == "SET" {
			key := parts[1]
			val := parts[2]
			ttl, _ := strconv.Atoi(parts[3])
			
			// Bypass AOF append since we are just replaying
			store.Set(key, val, ttl)
		}
	}

	// Seek back to end for future appends
	_, err = aof.file.Seek(0, 2)
	return err
}
