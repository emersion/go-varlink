package main

import (
	"log"
	"net"
	"syscall"

	"github.com/emersion/go-varlink"
	"github.com/emersion/go-varlink/example/internal/varlink/calcapi"
	"github.com/emersion/go-varlink/example/internal/varlink/stringapi"
)

type calcBackend struct{}

func (calcBackend) Multiply(in *calcapi.MultiplyIn) (*calcapi.MultiplyOut, error) {
	return &calcapi.MultiplyOut{Result: in.A * in.B}, nil
}

func (calcBackend) Divide(in *calcapi.DivideIn) (*calcapi.DivideOut, error) {
	if in.B == 0 {
		return nil, &calcapi.DivisionByZeroError{}
	}
	return &calcapi.DivideOut{Result: in.A / in.B}, nil
}

type stringBackend struct{}

func (stringBackend) Repeat(in *stringapi.RepeatIn) (*stringapi.RepeatOut, error) {
	return &stringapi.RepeatOut{Output: in.Input}, nil
}

func (stringBackend) Reverse(in *stringapi.ReverseIn) (*stringapi.ReverseOut, error) {
	result := make([]rune, len(in.Input))
	for i, char := range in.Input {
		result[len(in.Input)-i-1] = char
	}
	return &stringapi.ReverseOut{Output: string(result)}, nil
}

func main() {
	registry := varlink.NewRegistry(&varlink.RegistryOptions{
		Vendor:  "emersion/go-varlink",
		Product: "usage example",
		Version: "1.0",
		URL:     "https://github.com/emersion/go-varlink",
	})

	calcapi.Handler{Backend: calcBackend{}}.Register(registry)
	stringapi.Handler{Backend: stringBackend{}}.Register(registry)

	_ = syscall.Unlink("./org.example.sock")
	listener, err := net.Listen("unix", "./org.example.sock")
	if err != nil {
		log.Fatal(err.Error())
	}
	defer listener.Close()

	server := varlink.NewServer()
	server.Handler = registry
	if err = server.Serve(listener); err != nil {
		log.Fatal(err.Error())
	}
}
