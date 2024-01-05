// Package varlink implements the Varlink protocol.
//
// See https://varlink.org/
package varlink

import (
	"bufio"
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

	brw *bufio.ReadWriter
	enc *json.Encoder
	dec *json.Decoder
}

func newConn(c net.Conn) *conn {
	brw := &bufio.ReadWriter{
		Reader: bufio.NewReader(c),
		Writer: bufio.NewWriter(c),
	}
	return &conn{
		Conn: c,
		brw:  brw,
		enc:  json.NewEncoder(brw),
		dec:  json.NewDecoder(brw),
	}
}

func (c *conn) writeMessage(v interface{}) error {
	if err := c.enc.Encode(v); err != nil {
		return err
	}
	if _, err := c.brw.Write([]byte{0}); err != nil {
		return err
	}
	return c.brw.Flush()
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
