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

The generated file contains a `Client`, with one Go method per Varlink service
method:

```go
client := Client{varlink.NewClient(conn)}
_, err := client.Jump(&JumpIn{37.56, 126.99})
```

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

## License

MIT

[Varlink]: https://varlink.org/
