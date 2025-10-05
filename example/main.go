package main

import (
	"log"
	"net"
	"syscall"

	govarlink "github.com/emersion/go-varlink"
	"github.com/emersion/go-varlink/example/varlink/calcApi"
	"github.com/emersion/go-varlink/example/varlink/stringApi"
)

type calcBackend struct{}

func (calcBackend) Multiply(in *calcApi.MultiplyIn) (*calcApi.MultiplyOut, error) {
	return &calcApi.MultiplyOut{Result: in.A * in.B}, nil
}

func (calcBackend) Divide(in *calcApi.DivideIn) (*calcApi.DivideOut, error) {
	if in.B == 0 {
		return nil, &calcApi.DivisionByZeroError{}
	}
	return &calcApi.DivideOut{Result: in.A / in.B}, nil
}

type stringBackend struct{}

func (stringBackend) Repeat(in *stringApi.RepeatIn) (*stringApi.RepeatOut, error) {
	return &stringApi.RepeatOut{Output: in.Input}, nil
}

func (stringBackend) Reverse(in *stringApi.ReverseIn) (*stringApi.ReverseOut, error) {
	result := make([]rune, len(in.Input))
	for i, char := range in.Input {
		result[len(in.Input)-i-1] = char
	}
	return &stringApi.ReverseOut{Output: string(result)}, nil
}

func main() {
	registry := govarlink.NewRegistry()
	calcApi.Handler{Backend: calcBackend{}}.Register(registry)
	stringApi.Handler{Backend: stringBackend{}}.Register(registry)

	_ = syscall.Unlink("./org.example.sock")
	listener, err := net.Listen("unix", "./org.example.sock")
	if err != nil {
		log.Fatal(err.Error())
	}
	defer listener.Close()

	server := govarlink.NewServer()
	server.Handler = registry
	if err = server.Serve(listener); err != nil {
		log.Fatal(err.Error())
	}
}
