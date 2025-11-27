package varlink

import (
	_ "embed"
	"encoding/json"
)

//go:embed org.varlink.service.varlink
var definition string

// serviceHandler handles org.varlink.service introspection methods.
type serviceHandler struct {
	registry *Registry
}

type getInfoIn struct{}

// getInfoOut is returned by 'GetInfo', containing socket information.
type getInfoOut struct {
	Vendor     string   `json:"vendor"`
	Product    string   `json:"product"`
	Version    string   `json:"version"`
	URL        string   `json:"url"`
	Interfaces []string `json:"interfaces"`
}

// getInterfaceDescriptionIn is received during introspection.
type getInterfaceDescriptionIn struct {
	Interface string `json:"interface"`
}

// getInterfaceDescriptionOut is the introspection response,
// containing the string representation of the Varlink API.
type getInterfaceDescriptionOut struct {
	Description string `json:"description"`
}

type interfaceNotFoundError struct {
	Interface string `json:"interface"`
}

func (err *interfaceNotFoundError) Error() string {
	return "varlink call failed: org.varlink.service.InterfaceNotFound"
}

type invalidParameterError struct {
	Parameter string `json:"parameter"`
}

func (err *invalidParameterError) Error() string {
	return "varlink call failed: org.varlink.service.InvalidParameter"
}

type methodNotFoundError struct {
	Method string `json:"method"`
}

func (err *methodNotFoundError) Error() string {
	return "varlink call failed: org.varlink.service.MethodNotFound"
}

func marshalError(err error) error {
	var name string
	switch err.(type) {
	case *interfaceNotFoundError:
		name = "org.varlink.service.InterfaceNotFound"
	case *invalidParameterError:
		name = "org.varlink.service.InvalidParameter"
	case *methodNotFoundError:
		name = "org.varlink.service.MethodNotFound"
	default:
		return err
	}
	return &ServerError{
		Name:       name,
		Parameters: err,
	}
}

// HandleVarlink implements the Handler interface for org.varlink.service methods.
func (h *serviceHandler) HandleVarlink(call *ServerCall, req *ServerRequest) error {
	var (
		out interface{}
		err error
	)
	switch req.Method {
	case "org.varlink.service.GetInfo":
		out, err = h.GetInfo()
	case "org.varlink.service.GetInterfaceDescription":
		in := new(getInterfaceDescriptionIn)
		if err := json.Unmarshal(req.Parameters, in); err != nil {
			return err
		}
		out, err = h.GetInterfaceDescription(in)
	default:
		err = &methodNotFoundError{Method: req.Method}
	}
	if err != nil {
		return marshalError(err)
	}
	return call.CloseWithReply(out)
}

// GetInfo implements 'org.varlink.service.GetInfo' method, returning
// getInfoOut with socket information, and a list of implemented interfaces.
func (h *serviceHandler) GetInfo() (getInfoOut, error) {
	interfaces := make([]string, 0, len(h.registry.interfaces))
	for _, iface := range h.registry.interfaces {
		interfaces = append(interfaces, iface.Name)
	}

	return getInfoOut{
		Product:    h.registry.options.Product,
		Vendor:     h.registry.options.Vendor,
		Version:    h.registry.options.Version,
		URL:        h.registry.options.URL,
		Interfaces: interfaces,
	}, nil
}

// GetInterfaceDescription implements 'org.varlink.service.GetInterfaceDescription' method,
// returning getInterfaceDescriptionOut.
func (h *serviceHandler) GetInterfaceDescription(in *getInterfaceDescriptionIn) (*getInterfaceDescriptionOut, error) {
	iface, ok := h.registry.interfaces[in.Interface]
	if !ok {
		return nil, &interfaceNotFoundError{Interface: in.Interface}
	}

	return &getInterfaceDescriptionOut{Description: iface.Definition}, nil
}

func registerOrgVarlinkService(reg *Registry) {
	reg.Add(&RegistryInterface{
		Definition: definition,
		Name:       "org.varlink.service",
	}, &serviceHandler{registry: reg})
}
