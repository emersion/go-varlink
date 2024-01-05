// Package varlink implements the Varlink protocol.
//
// See https://varlink.org/
package varlink

import (
	"encoding/json"
	"fmt"
	"io"
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

	dec *json.Decoder
}

func newConn(c net.Conn) *conn {
	return &conn{
		Conn: c,
		dec:  json.NewDecoder(c),
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
	if err := c.dec.Decode(v); err != nil {
		return err
	}
	var b [1]byte
	if _, err := io.ReadFull(c.dec.Buffered(), b[:]); err != nil {
		return err
	} else if b[0] != 0 {
		return fmt.Errorf("varlink: expected NUL delimiter, got %v", b[0])
	}
	return nil
}
