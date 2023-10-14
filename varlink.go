// Package varlink implements the Varlink protocol.
//
// See https://varlink.org/
package varlink

import (
	"encoding/json"
	"fmt"
)

type Error struct {
	Name       string
	Parameters json.RawMessage
}

func (err *Error) Error() string {
	return fmt.Sprintf("varlink: request failed: %v", err.Name)
}
