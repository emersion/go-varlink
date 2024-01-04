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

type clientRequest struct {
	Method     string      `json:"method"`
	Parameters interface{} `json:"parameters"`
	Oneway     bool        `json:"oneway,omitempty"`
	More       bool        `json:"more,omitempty"`
	Upgrade    bool        `json:"upgrade,omitempty"`
}

type clientReply struct {
	Parameters json.RawMessage `json:"parameters"`
	Continues  bool            `json:"continues,omitempty"`
	Error      string          `json:"error,omitempty"`
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

func (c *Client) writeRequest(req *clientRequest, ch chan<- clientReply) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.err != nil {
		return c.err
	}

	c.pending = append(c.pending, ch)

	err := c.writeMessageLocked(req)
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
			if !reply.Continues {
				c.pending = c.pending[1:]
			}
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
	req := clientRequest{
		Method:     method,
		Parameters: in,
	}
	cc, err := c.do(&req)
	if err != nil {
		return err
	}
	continues, err := cc.next(out)
	if continues {
		c.conn.Close()
		return fmt.Errorf("varlink: received continues=true in response to a more=false request")
	}
	return err
}

func (c *Client) DoMore(method string, in interface{}) (*ClientCall, error) {
	req := clientRequest{
		Method:     method,
		Parameters: in,
		More:       true,
	}
	return c.do(&req)
}

func (c *Client) do(req *clientRequest) (*ClientCall, error) {
	if req.Parameters == nil {
		req.Parameters = struct{}{}
	}

	ch := make(chan clientReply, 32)
	if err := c.writeRequest(req, ch); err != nil {
		return nil, err
	}

	return &ClientCall{
		c:  c,
		ch: ch,
	}, nil
}

type ClientCall struct {
	c  *Client
	ch <-chan clientReply
}

func (cc *ClientCall) Next(out interface{}) error {
	if cc.ch == nil {
		return io.EOF
	}

	continues, err := cc.next(out)
	if !continues {
		cc.ch = nil
	}
	return err
}

func (cc *ClientCall) next(out interface{}) (continues bool, err error) {
	if out == nil {
		out = new(struct{})
	}

	reply, ok := <-cc.ch
	if !ok {
		return false, cc.c.err
	}

	if reply.Error != "" {
		return reply.Continues, &Error{Name: reply.Error, Parameters: reply.Parameters}
	}

	params := reply.Parameters
	if params == nil {
		params = json.RawMessage("{}")
	}
	return reply.Continues, json.Unmarshal(params, out)
}
