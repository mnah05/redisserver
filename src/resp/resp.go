package resp

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
)

// Parse reads from the reader and returns a Go native representation of the RESP data.
// It supports Simple Strings (+), Errors (-), Integers (:), Bulk Strings ($), Arrays (*), and Booleans (#).
func Parse(r *bufio.Reader) (any, error) {
	// Read the first byte to determine the type
	prefix, err := r.ReadByte()
	if err != nil {
		return nil, err
	}

	switch prefix {
	case '+': // Simple String
		return readLine(r)

	case '-': // Error
		// Return as error type or string, depending on preference.
		// Here we return a generic error with the message.
		line, err := readLine(r)
		if err != nil {
			return nil, err
		}
		return errors.New(string(line)), nil

	case ':': // Integer
		line, err := readLine(r)
		if err != nil {
			return nil, err
		}
		return strconv.ParseInt(string(line), 10, 64)

	case '#': // Boolean (RESP3)
		return readBoolean(r)

	case '$': // Bulk String
		return readBulkString(r)

	case '*': // Array
		return readArray(r)

	default:
		return nil, fmt.Errorf("unknown RESP prefix: %c", prefix)
	}
}

// readLine reads up to \r\n and strips the delimiter.
// It is stricter than strings.TrimSpace.
func readLine(r *bufio.Reader) ([]byte, error) {
	// ReadBytes includes the delimiter
	line, err := r.ReadBytes('\n')
	if err != nil {
		return nil, err
	}

	// RESP lines must end with \r\n
	if len(line) < 2 || line[len(line)-2] != '\r' {
		return nil, errors.New("line invalid: missing CRLF")
	}

	// Return content without \r\n
	return line[:len(line)-2], nil
}

func readBoolean(r *bufio.Reader) (bool, error) {
	line, err := readLine(r)
	if err != nil {
		return false, err
	}

	switch string(line) {
	case "t":
		return true, nil
	case "f":
		return false, nil
	default:
		return false, errors.New("invalid boolean value")
	}
}

func readBulkString(r *bufio.Reader) (string, error) {
	// 1. Read the length line
	line, err := readLine(r)
	if err != nil {
		return "", err
	}

	length, err := strconv.ParseInt(string(line), 10, 64)
	if err != nil {
		return "", err
	}

	// Handle Null Bulk String ($-1\r\n)
	if length == -1 {
		return "", nil // Or return a specific type indicating Null if needed
	}

	// Safety check: Avoid massive allocations
	// Redis strings max out at 512MB usually, but you might want a lower limit
	if length > 512*1024*1024 {
		return "", errors.New("bulk string too large")
	}

	// 2. Read the data
	buf := make([]byte, length)
	_, err = io.ReadFull(r, buf)
	if err != nil {
		return "", err
	}

	// 3. Consume the trailing \r\n
	// using ReadLine here to verify the CRLF exists
	if _, err := readLine(r); err != nil {
		return "", errors.New("bulk string missing trailing CRLF")
	}

	return string(buf), nil
}

func readArray(r *bufio.Reader) ([]any, error) {
	// 1. Read array length
	line, err := readLine(r)
	if err != nil {
		return nil, err
	}

	n, err := strconv.ParseInt(string(line), 10, 64)
	if err != nil {
		return nil, err
	}

	// Handle Null Array (*-1\r\n)
	if n == -1 {
		return nil, nil
	}

	// Safety check for array size
	if n > 1000000 {
		return nil, errors.New("array too large")
	}

	arr := make([]any, n)
	for i := range n {
		val, err := Parse(r)
		if err != nil {
			return nil, err
		}
		arr[i] = val
	}

	return arr, nil
}
