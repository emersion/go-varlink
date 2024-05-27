package varlink

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
)

// Edge represents a link between two nodes with a weight
type Edge struct {
	From, To int
	Weight   float64
}

// Graph represents the network topology
type Graph struct {
	Nodes int
	Edges []Edge
}

// TrafficMatrix represents the traffic matrix between nodes
type TrafficMatrix [][]float64

// Load represents the load on a link
type Load struct {
	Edge Edge
	Load float64
}

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

// ClientError is a Varlink error returned by a service to a Client.
type ClientError struct {
	Name       string
	Parameters json.RawMessage
}

// Error implements the error interface.
func (err *ClientError) Error() string {
	return fmt.Sprintf("varlink: client call failed: %v", err.Name)
}

// Client is a Varlink client.
//
// Client methods are safe to use from multiple goroutines.
type Client struct {
	conn *conn

	mutex   sync.Mutex
	pending []chan<- clientReply
	err     error
}

// NewClient creates a Varlink client from a net.Conn.
func NewClient(conn net.Conn) *Client {
	c := &Client{conn: newConn(conn)}
	go c.readLoop()
	return c
}

// Close closes the connection.
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

// Do performs a Varlink call.
//
// in is a Go value marshaled to a JSON object which contains the request
// parameters. Similarly, out will be populated with the reply parameters.
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

// DoMore is similar to Do, but indicates to the service that multiple replies
// are expected.
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

// ClientCall represents an in-progress Varlink method call.
type ClientCall struct {
	c  *Client
	ch <-chan clientReply
}

// Next waits for a reply.
//
// If there are no more replies, io.EOF is returned.
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

// New functionality for v0.2

// ReadTopology sends a request to read the topology file
func (c *Client) ReadTopology() (Graph, error) {
	var graph Graph
	err := c.Do("org.example.readTopology", nil, &graph)
	return graph, err
}

// ReadTrafficMatrix sends a request to read the traffic matrix
func (c *Client) ReadTrafficMatrix() (TrafficMatrix, error) {
	var matrix TrafficMatrix
	err := c.Do("org.example.readTrafficMatrix", nil, &matrix)
	return matrix, err
}

// ComputeLinkLoads sends a request to compute link loads based on the graph and traffic matrix
func (c *Client) ComputeLinkLoads(graph Graph, matrix TrafficMatrix) ([]Load, error) {
	body := map[string]interface{}{
		"graph":  graph,
		"matrix": matrix,
	}
	var loads []Load
	err := c.Do("org.example.computeLinkLoads", body, &loads)
	return loads, err
}

// OptimizeLinkWeights sends a request to optimize link weights based on the graph and traffic matrix
func (c *Client) OptimizeLinkWeights(graph Graph, matrix TrafficMatrix) ([]Load, error) {
	body := map[string]interface{}{
		"graph":  graph,
		"matrix": matrix,
	}
	var loads []Load
	err := c.Do("org.example.optimizeLinkWeights", body, &loads)
	return loads, err
}

func main() {
	// Establish connection to the server
	conn, err := net.Dial("unix", "/var/run/org.example")
	if err != nil {
		panic(err)
	}
	client := NewClient(conn)

	// Read topology and traffic matrix
	topology, err := client.ReadTopology()
	if err != nil {
		panic(err)
	}
	trafficMatrix, err := client.ReadTrafficMatrix()
	if err != nil {
		panic(err)
	}

	// Compute initial link loads
	linkLoads, err := client.ComputeLinkLoads(topology, trafficMatrix)
	if err != nil {
		panic(err)
	}
	fmt.Println("Initial link loads:")
	for _, load := range linkLoads {
		fmt.Printf("Link from %d to %d has load %f\n", load.Edge.From, load.Edge.To, load.Load)
	}

	// Optimize link weights
	optimizedLoads, err := client.OptimizeLinkWeights(topology, trafficMatrix)
	if err != nil {
		panic(err)
	}
	fmt.Println("Optimized link loads:")
	for _, load := range optimizedLoads {
		fmt.Printf("Link from %d to %d has load %f\n", load.Edge.From, load.Edge.To, load.Load)
	}

	// Close client connection
	client.Close()
}
