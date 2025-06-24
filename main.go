package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
)

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
	if msg == "" {
		return "$-1\r\n"
	}
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
			
		default:
			// Unknown command
			_, err = conn.Write([]byte(formatError("ERR unknown command")))
			if err != nil {
				fmt.Printf("Error writing unknown command response: %v\n", err)
				return
			}
		}
	}
}
