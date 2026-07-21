package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
)

var store *Store
var aof *AOF

func main() {
	fmt.Println("Starting Go Redis Clone on port 6380...")

	store = NewStore()
	
	// Setup Persistence
	var err error
	aof, err = NewAOF("database.aof")
	if err != nil {
		fmt.Printf("Error opening AOF: %v\n", err)
		return
	}
	defer aof.Close()

	fmt.Println("Replaying AOF...")
	aof.Replay(store)

	// Start TCP server
	listener, err := net.Listen("tcp", ":6380")
	if err != nil {
		fmt.Printf("Failed to bind to port 6380: %v\n", err)
		return
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)

	for {
		// Read a basic RESP command array, e.g. "*3\r\n$3\r\nSET\r\n$3\r\nfoo\r\n$3\r\nbar\r\n"
		line, err := reader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				fmt.Println("Error reading:", err)
			}
			return
		}

		// Simple inline command fallback (for telnet)
		if line[0] != '*' {
			parts := strings.Fields(line)
			if len(parts) > 0 {
				processCommand(parts, conn)
			}
			continue
		}

		// Parse RESP Array
		numArgs, _ := strconv.Atoi(strings.TrimSpace(line[1:]))
		args := make([]string, 0, numArgs)

		for i := 0; i < numArgs; i++ {
			// Read string length e.g. "$3"
			_, _ = reader.ReadString('\n')
			// Read actual string e.g. "SET"
			strLine, _ := reader.ReadString('\n')
			args = append(args, strings.TrimSpace(strLine))
		}

		if len(args) > 0 {
			processCommand(args, conn)
		}
	}
}

func processCommand(args []string, conn net.Conn) {
	cmd := strings.ToUpper(args[0])

	switch cmd {
	case "PING":
		conn.Write([]byte("+PONG\r\n"))
	case "SET":
		if len(args) < 3 {
			conn.Write([]byte("-ERR wrong number of arguments for 'set' command\r\n"))
			return
		}
		key := args[1]
		val := args[2]
		ttl := 0

		// Handle EX flag
		if len(args) >= 5 && strings.ToUpper(args[3]) == "EX" {
			parsedTTL, err := strconv.Atoi(args[4])
			if err == nil {
				ttl = parsedTTL
			}
		}

		store.Set(key, val, ttl)
		aof.AppendSet(key, val, ttl) // Persist
		conn.Write([]byte("+OK\r\n"))

	case "GET":
		if len(args) != 2 {
			conn.Write([]byte("-ERR wrong number of arguments for 'get' command\r\n"))
			return
		}
		val, ok := store.Get(args[1])
		if !ok {
			conn.Write([]byte("$-1\r\n")) // Null bulk string
		} else {
			conn.Write([]byte(fmt.Sprintf("$%d\r\n%s\r\n", len(val), val)))
		}

	case "COMMAND":
		// Ignore COMMAND docs request from clients
		conn.Write([]byte("+OK\r\n"))

	default:
		conn.Write([]byte(fmt.Sprintf("-ERR unknown command '%s'\r\n", cmd)))
	}
}
