package varlink

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
)

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
