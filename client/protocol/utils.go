package protocol

import (
	"encoding/binary"
	"fmt"
	"io"
)

// WriteString writes a string to an io.Writer with length prefix.
// The format is: 1 Byte for length + N Bytes for string content.
// Returns an error if the string exceeds 255 characters.
func WriteString(w io.Writer, s string) error {
	bytes := []byte(s)
	if len(bytes) > 255 {
		return fmt.Errorf("string field to be written is too long (255 chars max)")
	}
	if err := binary.Write(w, binary.BigEndian, uint8(len(bytes))); err != nil {
		return err
	}
	_, err := w.Write(bytes)
	return err
}

// ReadString reads a length-prefixed string from an io.Reader.
// The expected format is: 1 Byte for length + N Bytes for string content.
func ReadString(r io.Reader) (string, error) {
	var length uint8
	if err := binary.Read(r, binary.BigEndian, &length); err != nil {
		return "", err
	}
	bytes := make([]byte, length)
	if _, err := io.ReadFull(r, bytes); err != nil {
		return "", err
	}
	return string(bytes), nil
}
