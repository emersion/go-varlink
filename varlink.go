// Package varlink implements the Varlink protocol.
//
// See https://varlink.org/
package varlink

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
)

type Error struct {
	Name       string
	Parameters json.RawMessage
}

func (err *Error) Error() string {
	return fmt.Sprintf("varlink: request failed: %v", err.Name)
}

type conn struct {
	net.Conn

	br *bufio.Reader
}

func newConn(c net.Conn) *conn {
	return &conn{
		Conn: c,
		br:   bufio.NewReader(c),
	}
}

func (c *conn) writeMessage(v interface{}) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	b = append(b, 0)
	_, err = c.Write(b)
	return err
}

func (c *conn) readMessage(v interface{}) error {
	b, err := c.br.ReadBytes(0)
	if err != nil {
		return err
	}
	b = b[:len(b)-1]
	return json.Unmarshal(b, v)
}
