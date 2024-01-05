package varlink

import (
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

type ClientError struct {
	Name       string
	Parameters json.RawMessage
}

func (err *ClientError) Error() string {
	return fmt.Sprintf("varlink: client call failed: %v", err.Name)
}

type Client struct {
	conn *conn

	mutex   sync.Mutex
	pending []chan<- clientReply
	err     error
}

func NewClient(conn net.Conn) *Client {
	c := &Client{conn: newConn(conn)}
	go c.readLoop()
	return c
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) writeRequest(req *clientRequest, ch chan<- clientReply) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.err != nil {
		return c.err
	}

	c.pending = append(c.pending, ch)

	err := c.conn.writeMessage(req)
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
		if err = c.conn.readMessage(&reply); err != nil {
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
		return reply.Continues, &ClientError{Name: reply.Error, Parameters: reply.Parameters}
	}

	params := reply.Parameters
	if params == nil {
		params = json.RawMessage("{}")
	}
	return reply.Continues, json.Unmarshal(params, out)
}
