package varlinkservice

import (
	"encoding/json"
	govarlink "git.sr.ht/~emersion/go-varlink"
)

// Define errors
type ExpectedMoreError struct{}

func (err *ExpectedMoreError) Error() string {
	return "varlink call failed: org.varlink.service.ExpectedMore"
}

type InterfaceNotFoundError struct {
	Interface string `json:"interface"`
}

func (err *InterfaceNotFoundError) Error() string {
	return "varlink call failed: org.varlink.service.InterfaceNotFound"
}

type InvalidParameterError struct {
	Parameter string `json:"parameter"`
}

func (err *InvalidParameterError) Error() string {
	return "varlink call failed: org.varlink.service.InvalidParameter"
}

type MethodNotFoundError struct {
	Method string `json:"method"`
}

func (err *MethodNotFoundError) Error() string {
	return "varlink call failed: org.varlink.service.MethodNotFound"
}

type MethodNotImplementedError struct {
	Method string `json:"method"`
}

func (err *MethodNotImplementedError) Error() string {
	return "varlink call failed: org.varlink.service.MethodNotImplemented"
}

type PermissionDeniedError struct{}

func (err *PermissionDeniedError) Error() string {
	return "varlink call failed: org.varlink.service.PermissionDenied"
}

// Define input and output types
type GetInfoIn struct{}
type GetInfoOut struct {
	Interfaces []string `json:"interfaces"`
	Product    string   `json:"product"`
	Url        string   `json:"url"`
	Vendor     string   `json:"vendor"`
	Version    string   `json:"version"`
}

type GetInterfaceDescriptionIn struct {
	Interface string `json:"interface"`
}
type GetInterfaceDescriptionOut struct {
	Description string `json:"description"`
}

// Define types for new methods
type Graph struct {
	Nodes int    `json:"nodes"`
	Edges []Edge `json:"edges"`
}

type Edge struct {
	From   int     `json:"from"`
	To     int     `json:"to"`
	Weight float64 `json:"weight"`
}

type TrafficMatrix [][]float64

type Load struct {
	Edge Edge    `json:"edge"`
	Load float64 `json:"load"`
}

// Client struct
type Client struct {
	*govarlink.Client
}

func unmarshalError(err error) error {
	verr, ok := err.(*govarlink.ClientError)
	if !ok {
		return err
	}
	var v error
	switch verr.Name {
	case "org.varlink.service.ExpectedMore":
		v = new(ExpectedMoreError)
	case "org.varlink.service.InterfaceNotFound":
		v = new(InterfaceNotFoundError)
	case "org.varlink.service.InvalidParameter":
		v = new(InvalidParameterError)
	case "org.varlink.service.MethodNotFound":
		v = new(MethodNotFoundError)
	case "org.varlink.service.MethodNotImplemented":
		v = new(MethodNotImplementedError)
	case "org.varlink.service.PermissionDenied":
		v = new(PermissionDeniedError)
	default:
		return err
	}
	if err := json.Unmarshal(verr.Parameters, v); err != nil {
		return err
	}
	return v
}

func (c Client) GetInfo(in *GetInfoIn) (*GetInfoOut, error) {
	if in == nil {
		in = new(GetInfoIn)
	}
	out := new(GetInfoOut)
	err := c.Client.Do("org.varlink.service.GetInfo", in, out)
	return out, unmarshalError(err)
}

func (c Client) GetInterfaceDescription(in *GetInterfaceDescriptionIn) (*GetInterfaceDescriptionOut, error) {
	if in == nil {
		in = new(GetInterfaceDescriptionIn)
	}
	out := new(GetInterfaceDescriptionOut)
	err := c.Client.Do("org.varlink.service.GetInterfaceDescription", in, out)
	return out, unmarshalError(err)
}

// Define new methods for the client
func (c Client) ReadTopology() (*Graph, error) {
	var out Graph
	err := c.Client.Do("org.example.readTopology", nil, &out)
	return &out, unmarshalError(err)
}

func (c Client) ReadTrafficMatrix() (*TrafficMatrix, error) {
	var out TrafficMatrix
	err := c.Client.Do("org.example.readTrafficMatrix", nil, &out)
	return &out, unmarshalError(err)
}

func (c Client) ComputeLinkLoads(graph Graph, matrix TrafficMatrix) ([]Load, error) {
	in := struct {
		Graph  Graph         `json:"graph"`
		Matrix TrafficMatrix `json:"matrix"`
	}{graph, matrix}
	var out []Load
	err := c.Client.Do("org.example.computeLinkLoads", in, &out)
	return out, unmarshalError(err)
}

func (c Client) OptimizeLinkWeights(graph Graph, matrix TrafficMatrix) ([]Load, error) {
	in := struct {
		Graph  Graph         `json:"graph"`
		Matrix TrafficMatrix `json:"matrix"`
	}{graph, matrix}
	var out []Load
	err := c.Client.Do("org.example.optimizeLinkWeights", in, &out)
	return out, unmarshalError(err)
}

// Backend interface
type Backend interface {
	GetInfo(*GetInfoIn) (*GetInfoOut, error)
	GetInterfaceDescription(*GetInterfaceDescriptionIn) (*GetInterfaceDescriptionOut, error)
	ReadTopology() (*Graph, error)
	ReadTrafficMatrix() (*TrafficMatrix, error)
	ComputeLinkLoads(graph Graph, matrix TrafficMatrix) ([]Load, error)
	OptimizeLinkWeights(graph Graph, matrix TrafficMatrix) ([]Load, error)
}

// Handler struct
type Handler struct {
	Backend Backend
}

func marshalError(err error) error {
	var name string
	switch err.(type) {
	case *ExpectedMoreError:
		name = "org.varlink.service.ExpectedMore"
	case *InterfaceNotFoundError:
		name = "org.varlink.service.InterfaceNotFound"
	case *InvalidParameterError:
		name = "org.varlink.service.InvalidParameter"
	case *MethodNotFoundError:
		name = "org.varlink.service.MethodNotFound"
	case *MethodNotImplementedError:
		name = "org.varlink.service.MethodNotImplemented"
	case *PermissionDeniedError:
		name = "org.varlink.service.PermissionDenied"
	default:
		return err
	}
	return &govarlink.ServerError{
		Name:       name,
		Parameters: err,
	}
}

func (h Handler) HandleVarlink(call *govarlink.ServerCall, req *govarlink.ServerRequest) error {
	var (
		out interface{}
		err error
	)
	switch req.Method {
	case "org.varlink.service.GetInfo":
		in := new(GetInfoIn)
		if err := json.Unmarshal(req.Parameters, in); err != nil {
			return err
		}
		out, err = h.Backend.GetInfo(in)
	case "org.varlink.service.GetInterfaceDescription":
		in := new(GetInterfaceDescriptionIn)
		if err := json.Unmarshal(req.Parameters, in); err != nil {
			return err
		}
		out, err = h.Backend.GetInterfaceDescription(in)
	case "org.example.readTopology":
		out, err = h.Backend.ReadTopology()
	case "org.example.readTrafficMatrix":
		out, err = h.Backend.ReadTrafficMatrix()
	case "org.example.computeLinkLoads":
		in := struct {
			Graph  Graph         `json:"graph"`
			Matrix TrafficMatrix `json:"matrix"`
		}{}
		if err := json.Unmarshal(req.Parameters, &in); err != nil {
			return err
		}
		out, err = h.Backend.ComputeLinkLoads(in.Graph, in.Matrix)
	case "org.example.optimizeLinkWeights":
		in := struct {
			Graph  Graph         `json:"graph"`
			Matrix TrafficMatrix `json:"matrix"`
		}{}
		if err := json.Unmarshal(req.Parameters, &in); err != nil {
			return err
		}
		out, err = h.Backend.OptimizeLinkWeights(in.Graph, in.Matrix)
	default:
		err = &govarlink.ServerError{
			Name:       "org.varlink.service.MethodNotFound",
			Parameters: map[string]string{"method": req.Method},
		}
	}
	if err != nil {
		return marshalError(err)
	}
	return call.CloseWithReply(out)
}
