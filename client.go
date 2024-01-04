package varlink

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
)

type clientReply struct {
	Parameters json.RawMessage `json:"parameters"`
	Error      string          `json:"error"`
}

type Client struct {
	conn net.Conn

	mutex   sync.Mutex
	brw     *bufio.ReadWriter
	enc     *json.Encoder
	dec     *json.Decoder
	pending []chan<- clientReply
	err     error
}

func NewClient(conn net.Conn) *Client {
	brw := &bufio.ReadWriter{
		Reader: bufio.NewReader(conn),
		Writer: bufio.NewWriter(conn),
	}
	c := &Client{
		conn: conn,
		brw:  brw,
		enc:  json.NewEncoder(brw),
		dec:  json.NewDecoder(brw),
	}
	go c.readLoop()
	return c
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) writeMessageLocked(v interface{}) error {
	if err := c.enc.Encode(v); err != nil {
		return err
	}
	if _, err := c.brw.Write([]byte{0}); err != nil {
		return err
	}
	return c.brw.Flush()
}

func (c *Client) readMessage(v interface{}) error {
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

func (c *Client) writeRequest(method string, parameters interface{}, ch chan<- clientReply) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.err != nil {
		return c.err
	}

	c.pending = append(c.pending, ch)

	err := c.writeMessageLocked(map[string]interface{}{
		"method":     method,
		"parameters": parameters,
	})
	if err != nil {
		c.err = err
		c.conn.Close()
		return err
	}

	return nil
}

func (c *Client) readLoop() {
	var err error
	defer func() {
		c.mutex.Lock()
		defer c.mutex.Unlock()

		if err != nil {
			c.err = err
		}

		for _, ch := range c.pending {
			close(ch)
		}
		c.pending = nil
	}()

	for {
		var reply clientReply
		if err = c.readMessage(&reply); err != nil {
			if errors.Is(err, net.ErrClosed) {
				err = nil
			}
			break
		}

		var ch chan<- clientReply
		c.mutex.Lock()
		if len(c.pending) > 0 {
			ch = c.pending[0]
			c.pending = c.pending[1:]
		}
		c.mutex.Unlock()

		if ch == nil {
			err = fmt.Errorf("varlink: received reply without request")
			break
		}

		ch <- reply
	}
}

func (c *Client) Do(method string, in, out interface{}) error {
	if in == nil {
		in = struct{}{}
	}
	if out == nil {
		out = new(struct{})
	}

	ch := make(chan clientReply, 1)
	if err := c.writeRequest(method, in, ch); err != nil {
		return err
	}

	reply, ok := <-ch
	if !ok {
		return c.err
	}

	if reply.Error != "" {
		return &Error{Name: reply.Error, Parameters: reply.Parameters}
	}

	params := reply.Parameters
	if params == nil {
		params = json.RawMessage("{}")
	}
	return json.Unmarshal(params, out)
}
