//go:build generate

package main

import (
	_ "github.com/emersion/go-varlink/cmd/varlinkgen"
)

//go:generate go run github.com/emersion/go-varlink/cmd/varlinkgen -i internal/varlink/calcapi/org.example.calc.varlink
//go:generate go run github.com/emersion/go-varlink/cmd/varlinkgen -i internal/varlink/stringapi/org.example.string.varlink
