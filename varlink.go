// Package varlink implements the Varlink protocol.
//
// See https://varlink.org/
package varlink

import (
	"bufio"
	"encoding/json"
	"net"
)

// conn represents a Varlink connection.
type conn struct {
	net.Conn
	br *bufio.Reader
}

// newConn creates a new Varlink connection.
func newConn(c net.Conn) *conn {
	return &conn{
		Conn: c,
		br:   bufio.NewReader(c),
	}
}

// writeMessage writes a Varlink message to the connection.
func (c *conn) writeMessage(v interface{}) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	b = append(b, 0) // Append a null byte as Varlink message delimiter.
	_, err = c.Write(b)
	return err
}

// readMessage reads a Varlink message from the connection.
func (c *conn) readMessage(v interface{}) error {
	b, err := c.br.ReadBytes(0)
	if err != nil {
		return err
	}
	b = b[:len(b)-1] // Remove the null byte delimiter.
	return json.Unmarshal(b, v)
}
