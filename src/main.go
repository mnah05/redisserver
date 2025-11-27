package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"sync"

	"github.com/mnah05/redisserver/src/resp"
)

var (
	// store holds the Key-Value data
	store = make(map[string]string)
	// mu protects the store from concurrent writes
	mu sync.RWMutex
)

func main() {
	// 1. Setup Logging
	log.Println("Server is listening on port :6379")

	l, err := net.Listen("tcp", ":6379")
	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()

	for {
		// 2. Accept new connection
		conn, err := l.Accept()
		if err != nil {
			log.Println("Error accepting connection:", err)
			continue
		}

		// 3. Handle connection in a separate Goroutine
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	// Log who connected
	clientAddr := conn.RemoteAddr().String()
	log.Printf("[%s] Client connected", clientAddr)

	reader := bufio.NewReader(conn)

	for {
		value, err := resp.Parse(reader)
		if err != nil {
			if err == io.EOF {
				log.Printf("[%s] Client disconnected", clientAddr)
				break
			}
			log.Printf("[%s] Parse error: %v", clientAddr, err)
			return
		}

		// Convert parsed value to a slice of commands
		commandParts, ok := value.([]any)
		if !ok || len(commandParts) == 0 {
			continue
		}

		cmdName, _ := commandParts[0].(string)
		cmdName = strings.ToUpper(cmdName)

		log.Printf("[%s] Command: %v", clientAddr, commandParts)

		switch cmdName {
		case "PING":
			conn.Write([]byte("+PONG\r\n"))

		case "SET":
			if len(commandParts) != 3 {
				conn.Write([]byte("-ERR wrong number of arguments for 'set'\r\n"))
				continue
			}
			key, _ := commandParts[1].(string)
			val, _ := commandParts[2].(string)

			mu.Lock() 
			store[key] = val
			mu.Unlock()

			conn.Write([]byte("+OK\r\n"))

		case "GET":
			if len(commandParts) != 2 {
				conn.Write([]byte("-ERR wrong number of arguments for 'get'\r\n"))
				continue
			}
			key, _ := commandParts[1].(string)

			mu.RLock() 
			val, exists := store[key]
			mu.RUnlock()

			if !exists {
				conn.Write([]byte("$-1\r\n"))
			} else {
				res := fmt.Sprintf("$%d\r\n%s\r\n", len(val), val)
				conn.Write([]byte(res))
			}

		default:
			conn.Write([]byte("-ERR unknown command\r\n"))
		}
	}
}
