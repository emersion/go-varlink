package varlink

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
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

// ServerRequest is a request coming from a Varlink client.
type ServerRequest struct {
	Method     string          `json:"method"`
	Parameters json.RawMessage `json:"parameters"`
	Oneway     bool            `json:"oneway,omitempty"`
	More       bool            `json:"more,omitempty"`
	Upgrade    bool            `json:"upgrade,omitempty"`
}

type serverReply struct {
	Parameters interface{} `json:"parameters"`
	Continues  bool        `json:"continues,omitempty"`
	Error      string      `json:"error,omitempty"`
}

// ServerError is an error to be sent to a Varlink client.
type ServerError struct {
	Name       string
	Parameters interface{}
}

// Error implements the error interface.
func (err *ServerError) Error() string {
	return fmt.Sprintf("varlink: server call failed: %v", err.Name)
}

// ServerCall represents an in-progress Varlink method call.
//
// Handlers may call Reply any number of times, then they must end the call
// with CloseWithReply.
type ServerCall struct {
	conn *conn
	req  *ServerRequest
	done bool
}

func (call *ServerCall) reply(reply *serverReply) error {
	if reply.Continues {
		if !call.req.More {
			return fmt.Errorf("varlink: ServerCall.Reply called for a request without More set")
		}
	} else {
		if call.done {
			return fmt.Errorf("varlink: ServerCall.CloseWithReply called twice")
		}
		call.done = true
	}
	if call.req.Oneway {
		return nil
	}
	return call.conn.writeMessage(reply)
}

// Reply sends a non-final reply.
//
// This can only be used if ServerRequest.More is set to true.
func (call *ServerCall) Reply(parameters interface{}) error {
	return call.reply(&serverReply{
		Parameters: parameters,
		Continues:  true,
	})
}

// CloseWithReply sends a final reply and closes the call.
//
// No more replies may be sent.
func (call *ServerCall) CloseWithReply(parameters interface{}) error {
	return call.reply(&serverReply{Parameters: parameters})
}

// A Handler processes Varlink requests.
type Handler interface {
	HandleVarlink(call *ServerCall, req *ServerRequest) error
}

// Server is a Varlink server.
//
// The Handler field must be set to a Varlink request handler.
type Server struct {
	Handler Handler
}

// NewServer creates a new Varlink server.
func NewServer() *Server {
	return &Server{}
}

// Serve listens for connections.
func (srv *Server) Serve(ln net.Listener) error {
	for {
		conn, err := ln.Accept()
		if err != nil {
			return err
		}
		go func() {
			if err := srv.serveConn(newConn(conn)); err != nil {
				log.Printf("varlink: serving connection: %v", err)
			}
		}()
	}
}

func (srv *Server) serveConn(conn *conn) error {
	defer conn.Close()

	for {
		var req ServerRequest
		if err := conn.readMessage(&req); err == io.EOF {
			return nil
		} else if err != nil {
			return fmt.Errorf("reading request: %v", err)
		}

		if req.Upgrade {
			return fmt.Errorf("varlink: connection upgrades not implemented")
		}

		call := &ServerCall{
			conn: conn,
			req:  &req,
		}
		err := srv.Handler.HandleVarlink(call, &req)
		var verr *ServerError
		if errors.As(err, &verr) {
			if req.Oneway {
				continue
			}
			if err := call.reply(&serverReply{
				Error:      verr.Name,
				Parameters: verr.Parameters,
			}); err != nil {
				return fmt.Errorf("writing error: %v", err)
			}
		} else if err != nil {
			return fmt.Errorf("handling call: %v", err)
		}

		if !req.Oneway && !call.done {
			return fmt.Errorf("varlink: ServerCall.CloseWithReply not called")
		}
	}
}

// New functionality for v0.2

// ReadTopology reads the topology file and returns a Graph
func ReadTopology() (Graph, error) {
	data, err := ioutil.ReadFile("topology.json")
	if err != nil {
		return Graph{}, err
	}
	var graph Graph
	err = json.Unmarshal(data, &graph)
	return graph, err
}

// ReadTrafficMatrix reads the traffic matrix file and returns a TrafficMatrix
func ReadTrafficMatrix() (TrafficMatrix, error) {
	data, err := ioutil.ReadFile("traffic_matrix.json")
	if err != nil {
		return nil, err
	}
	var matrix TrafficMatrix
	err = json.Unmarshal(data, &matrix)
	return matrix, err
}

// ComputeLinkLoads computes the load on each link based on the traffic matrix
func ComputeLinkLoads(graph Graph, matrix TrafficMatrix) []Load {
	loads := make(map[Edge]float64)

	for i, row := range matrix {
		for j, traffic := range row {
			if i != j && traffic > 0 {
				for _, edge := range graph.Edges {
					if (edge.From == i && edge.To == j) || (edge.From == j && edge.To == i) {
						loads[edge] += traffic
					}
				}
			}
		}
	}

	var linkLoads []Load
	for edge, load := range loads {
		linkLoads = append(linkLoads, Load{Edge: edge, Load: load})
	}

	return linkLoads
}

// LocalSearchHeuristic optimizes the link weights using a local search heuristic
func LocalSearchHeuristic(graph *Graph, matrix TrafficMatrix, maxIterations int) []Load {
	bestGraph := *graph
	bestLoad := ComputeLinkLoads(bestGraph, matrix)
	bestCost := CalculateCost(bestLoad)

	for i := 0; i < maxIterations; i++ {
		newGraph := PerturbGraph(bestGraph)
		newLoad := ComputeLinkLoads(newGraph, matrix)
		newCost := CalculateCost(newLoad)

		if newCost < bestCost {
			bestGraph = newGraph
			bestLoad = newLoad
			bestCost = newCost
		}
	}

	return bestLoad
}

// PerturbGraph randomly adjusts the weights of the edges in the graph
func PerturbGraph(graph Graph) Graph {
	newGraph := graph
	for i := range newGraph.Edges {
		newGraph.Edges[i].Weight += float64(rand.Intn(10) - 5)
	}
	return newGraph
}

// CalculateCost calculates the total load of the network
func CalculateCost(loads []Load) float64 {
	totalLoad := 0.0
	for _, load := range loads {
		totalLoad += load.Load
	}
	return totalLoad
}

// VarlinkHandler is the struct that implements the Handler interface
type VarlinkHandler struct{}

// HandleVarlink handles incoming Varlink requests
func (h *VarlinkHandler) HandleVarlink(call *ServerCall, req *ServerRequest) error {
	switch req.Method {
	case "org.example.readTopology":
		graph, err := ReadTopology()
		if err != nil {
			return &ServerError{Name: "org.example.readTopologyFailed", Parameters: err.Error()}
		}
		return call.CloseWithReply(graph)
	case "org.example.readTrafficMatrix":
		matrix, err := ReadTrafficMatrix()
		if err != nil {
			return &ServerError{Name: "org.example.readTrafficMatrixFailed", Parameters: err.Error()}
		}
		return call.CloseWithReply(matrix)
	case "org.example.computeLinkLoads":
		var input struct {
			Graph  Graph         `json:"graph"`
			Matrix TrafficMatrix `json:"matrix"`
		}
		if err := json.Unmarshal(req.Parameters, &input); err != nil {
			return &ServerError{Name: "org.example.invalidParameters", Parameters: err.Error()}
		}
		loads := ComputeLinkLoads(input.Graph, input.Matrix)
		return call.CloseWithReply(loads)
	case "org.example.optimizeLinkWeights":
		var input struct {
			Graph  Graph         `json:"graph"`
			Matrix TrafficMatrix `json:"matrix"`
		}
		if err := json.Unmarshal(req.Parameters, &input); err != nil {
			return &ServerError{Name: "org.example.invalidParameters", Parameters: err.Error()}
		}
		optimizedLoads := LocalSearchHeuristic(&input.Graph, input.Matrix, 1000) // Assuming 1000 iterations
		return call.CloseWithReply(optimizedLoads)
	default:
		return &ServerError{Name: "org.example.methodNotFound", Parameters: req.Method}
	}
}

func main() {
	// Create a new Varlink server
	server := NewServer()
	server.Handler = &VarlinkHandler{}

	// Listen on a Unix socket
	ln, err := net.Listen("unix", "/var/run/org.example")
	if err != nil {
		log.Fatalf("failed to listen on socket: %v", err)
	}
	defer ln.Close()

	log.Println("Server is running...")
	if err := server.Serve(ln); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
