# go-varlink

[![Go Reference](https://pkg.go.dev/badge/github.com/emersion/go-varlink.svg)](https://pkg.go.dev/github.com/emersion/go-varlink)

A Go library for [Varlink].

## Code generation

Given a Varlink definition file:

```varlink
interface org.example.ftl

method Jump(latitude: float, longitude: float) -> ()
```

Client and server code can be generated:

    go run github.com/emersion/go-varlink/cmd/varlinkgen -i org.example.ftl.varlink

This can be performed with `go generate`:

```go
//go:build generate

package ftl

import (
	_ "github.com/emersion/go-varlink/cmd/varlinkgen"
)

//go:generate go run github.com/emersion/go-varlink/cmd/varlinkgen -i org.example.ftl.varlink
```

## Client

The generated file contains a `Client`, with one Go method per Varlink service
method:

```go
client := Client{varlink.NewClient(conn)}
_, err := client.Jump(&JumpIn{37.56, 126.99})
```

## Server

It also contains a `Handler` implementing the Varlink service, and a `Backend`
interface which needs to be implemented:

```go
type backend struct{}

func (backend) Jump(in *JumpIn) (*JumpOut, error) {
    log.Print(in.Latitude, in.Longitude)
    return nil, nil
}

func main() {
    server := varlink.NewServer()
    server.Handler = Handler{backend{}}
    if err := server.Serve(listener); err != nil {
        log.Fatal(err)
    }
}
```

### Registry

`Registry` can act as a dispatcher to route requests to appropriate backends:

```go
type backendA struct{}
type backendB struct{}

func main() {
    registry := govarlink.NewRegistry(&govarlink.RegistryOptions{
        Vendor:  "my-vendor",
        Product: "my-product",
        Version: "1.0",
        URL:     "https://example.com",
    })
    aApi.Handler{Backend: backendA{}}.Register(registry)
    bApi.Handler{Backend: backendB{}}.Register(registry)

    server := govarlink.NewServer()
    server.Handler = registry
    if err := server.Serve(listener); err != nil {
        log.Fatal(err.Error())
    }
}
```

## License

MIT

[Varlink]: https://varlink.org/
