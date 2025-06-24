package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Data type constants (string identifiers)
const (
	TypeString    = "string"
	TypeList      = "list"
	TypeSet       = "set"
	TypeHash      = "hash"
	TypeSortedSet = "sortedset"
)

// Data type constants (numeric identifiers for future optimization)
const (
	TypeStringID    = 1
	TypeListID      = 2
	TypeSetID       = 3
	TypeHashID      = 4
	TypeSortedSetID = 5
)

// Entry represents a single key-value entry in the store
type Entry struct {
	Type      string      // Data type (string, list, set, hash, sortedset)
	Value     interface{} // Actual data (cast based on Type)
	ExpiresAt time.Time   // TTL expiration time (zero value means no expiration)
}

// Store represents the in-memory database
type Store struct {
	data map[string]*Entry // Key-value storage
	mu   sync.RWMutex      // Read-write mutex for synchronization
}

// NewStore creates and initializes a new Store instance
func NewStore() *Store {
	return &Store{
		data: make(map[string]*Entry),
	}
}

// Get retrieves a string value for the given key
// Returns (value, exists, isCorrectType)
func (s *Store) Get(key string) (string, bool, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	entry, exists := s.data[key]
	if !exists {
		return "", false, true // Key doesn't exist, but type would be correct
	}
	
	// Check if the entry is of string type
	if entry.Type != TypeString {
		return "", true, false // Key exists but wrong type
	}
	
	// Retrieve the string value
	value, ok := entry.Value.(string)
	if !ok {
		return "", true, false // Type assertion failed
	}
	
	return value, true, true
}

// Set stores a string value for the given key
func (s *Store) Set(key string, val string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Create a new entry of string type
	entry := &Entry{
		Type:      TypeString,
		Value:     val,
		ExpiresAt: time.Time{}, // No expiration (zero value)
	}
	
	// Insert or replace the entry
	s.data[key] = entry
	
	return "OK"
}

// Del deletes one or more keys and returns the count of deleted keys
func (s *Store) Del(keys ...string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	count := 0
	for _, key := range keys {
		if _, exists := s.data[key]; exists {
			delete(s.data, key)
			count++
		}
	}
	
	return count
}

// Helper method to check if a key exists and get its type
func (s *Store) KeyType(key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	entry, exists := s.data[key]
	if !exists {
		return "", false
	}
	return entry.Type, true
}

// Helper method to set a non-string value for testing WRONGTYPE scenarios
func (s *Store) SetForTesting(key string, entryType string, value interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	entry := &Entry{
		Type:      entryType,
		Value:     value,
		ExpiresAt: time.Time{},
	}
	s.data[key] = entry
}

// Global store instance
var store = NewStore()

func main() {
	listener, err := net.Listen("tcp", ":6379")
	if err != nil {
		fmt.Printf("Error starting TCP server: %v\n", err)
		return
	}
	defer listener.Close()
	fmt.Println("Server is listening on port 6379")

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Printf("Error accepting connection: %v\n", err)
			continue
		}

		fmt.Println("New client connected")
		go handleConnection(conn)
	}
}

// ReadRESP reads the next RESP message from the reader
func ReadRESP(reader *bufio.Reader) ([]string, error) {
	firstByte, err := reader.ReadByte()
	if err != nil {
		return nil, err
	}

	switch firstByte {
	case '*': // Array
		// Read array length
		lengthStr, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		lengthStr = strings.TrimSuffix(lengthStr, "\r\n")
		length, err := strconv.Atoi(lengthStr)
		if err != nil {
			return nil, errors.New("invalid array length")
		}

		// Read array elements
		result := make([]string, length)
		for i := 0; i < length; i++ {
			// Expect bulk string
			dollarByte, err := reader.ReadByte()
			if err != nil {
				return nil, err
			}
			if dollarByte != '$' {
				return nil, errors.New("expected bulk string in array")
			}

			// Read bulk string length
			bulkLengthStr, err := reader.ReadString('\n')
			if err != nil {
				return nil, err
			}
			bulkLengthStr = strings.TrimSuffix(bulkLengthStr, "\r\n")
			bulkLength, err := strconv.Atoi(bulkLengthStr)
			if err != nil {
				return nil, errors.New("invalid bulk string length")
			}

			if bulkLength == -1 {
				// Null bulk string
				result[i] = ""
			} else {
				// Read bulk string data
				data := make([]byte, bulkLength)
				_, err := io.ReadFull(reader, data)
				if err != nil {
					return nil, err
				}
				result[i] = string(data)

				// Consume trailing CRLF
				trailing := make([]byte, 2)
				_, err = io.ReadFull(reader, trailing)
				if err != nil {
					return nil, err
				}
				if string(trailing) != "\r\n" {
					return nil, errors.New("expected CRLF after bulk string")
				}
			}
		}
		return result, nil

	case '+': // Simple String
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimSuffix(line, "\r\n")
		return []string{line}, nil

	case '-': // Error
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimSuffix(line, "\r\n")
		return []string{"ERROR", line}, nil

	case ':': // Integer
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimSuffix(line, "\r\n")
		return []string{"INTEGER", line}, nil

	case '$': // Bulk String (standalone)
		// Read bulk string length
		bulkLengthStr, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		bulkLengthStr = strings.TrimSuffix(bulkLengthStr, "\r\n")
		bulkLength, err := strconv.Atoi(bulkLengthStr)
		if err != nil {
			return nil, errors.New("invalid bulk string length")
		}

		if bulkLength == -1 {
			// Null bulk string
			return []string{""}, nil
		}

		// Read bulk string data
		data := make([]byte, bulkLength)
		_, err = io.ReadFull(reader, data)
		if err != nil {
			return nil, err
		}

		// Consume trailing CRLF
		trailing := make([]byte, 2)
		_, err = io.ReadFull(reader, trailing)
		if err != nil {
			return nil, err
		}
		if string(trailing) != "\r\n" {
			return nil, errors.New("expected CRLF after bulk string")
		}

		return []string{string(data)}, nil

	default:
		// Handle inline commands (like PING without RESP formatting)
		// Put the byte back and read as simple line
		err = reader.UnreadByte()
		if err != nil {
			return nil, err
		}
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			return nil, errors.New("empty command")
		}
		return strings.Fields(line), nil
	}
}

// RESP response formatting functions
func formatSimpleString(msg string) string {
	return "+" + msg + "\r\n"
}

func formatError(msg string) string {
	return "-" + msg + "\r\n"
}

func formatInteger(val int) string {
	return ":" + strconv.Itoa(val) + "\r\n"
}

func formatBulkString(msg string) string {
	return "$" + strconv.Itoa(len(msg)) + "\r\n" + msg + "\r\n"
}

func formatArray(elems []string) string {
	result := "*" + strconv.Itoa(len(elems)) + "\r\n"
	for _, elem := range elems {
		result += formatBulkString(elem)
	}
	return result
}

func handleConnection(conn net.Conn) {
	defer func() {
		conn.Close()
		fmt.Println("Client disconnected")
	}()

	reader := bufio.NewReader(conn)
	for {
		cmdParts, err := ReadRESP(reader)
		if err != nil {
			if err == io.EOF {
				return
			}
			// Protocol error
			_, writeErr := conn.Write([]byte(formatError("ERR Protocol error")))
			if writeErr != nil {
				fmt.Printf("Error writing protocol error response: %v\n", writeErr)
			}
			return
		}

		if len(cmdParts) == 0 {
			continue
		}

		// Normalize command name to uppercase for case-insensitivity
		command := strings.ToUpper(cmdParts[0])
		
		switch command {
		case "PING":
			// If an argument is provided, use it as message, else default "PONG"
			message := "PONG"
			if len(cmdParts) > 1 {
				message = cmdParts[1]
			}
			_, err = conn.Write([]byte(formatSimpleString(message)))
			if err != nil {
				fmt.Printf("Error writing PING response: %v\n", err)
				return
			}
			
		case "ECHO":
			// Respond with the argument as a bulk string
			if len(cmdParts) < 2 {
				_, err = conn.Write([]byte(formatError("ERR wrong number of arguments for 'echo' command")))
			} else {
				_, err = conn.Write([]byte(formatBulkString(cmdParts[1])))
			}
			if err != nil {
				fmt.Printf("Error writing ECHO response: %v\n", err)
				return
			}
			
		case "GET":
			// GET key
			if len(cmdParts) != 2 {
				_, err = conn.Write([]byte(formatError("ERR wrong number of arguments for 'get' command")))
			} else {
				key := cmdParts[1]
				value, exists, isCorrectType := store.Get(key)
				
				if exists && !isCorrectType {
					// Key exists but wrong type
					_, err = conn.Write([]byte(formatError("WRONGTYPE Operation against a key holding the wrong kind of value")))
				} else if !exists {
					// Key not found - return nil bulk string
					_, err = conn.Write([]byte("$-1\r\n"))
				} else {
					// Key found and correct type
					_, err = conn.Write([]byte(formatBulkString(value)))
				}
			}
			if err != nil {
				fmt.Printf("Error writing GET response: %v\n", err)
				return
			}
			
		case "SET":
			// SET key value
			if len(cmdParts) != 3 {
				_, err = conn.Write([]byte(formatError("ERR wrong number of arguments for 'set' command")))
			} else {
				key := cmdParts[1]
				value := cmdParts[2]
				result := store.Set(key, value)
				_, err = conn.Write([]byte(formatSimpleString(result)))
			}
			if err != nil {
				fmt.Printf("Error writing SET response: %v\n", err)
				return
			}
			
		case "DEL":
			// DEL key [key ...]
			if len(cmdParts) < 2 {
				_, err = conn.Write([]byte(formatError("ERR wrong number of arguments for 'del' command")))
			} else {
				keys := cmdParts[1:] // All arguments after command name
				count := store.Del(keys...)
				_, err = conn.Write([]byte(formatInteger(count)))
			}
			if err != nil {
				fmt.Printf("Error writing DEL response: %v\n", err)
				return
			}
			
		default:
			// Unknown command
			_, err = conn.Write([]byte(formatError(fmt.Sprintf("ERR unknown command '%s'", strings.ToLower(command)))))
			if err != nil {
				fmt.Printf("Error writing unknown command response: %v\n", err)
				return
			}
		}
	}
}
