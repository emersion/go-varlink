//go:build generate

package varlinkservice

import (
	_ "github.com/emersion/go-varlink/cmd/varlinkgen"
)

//go:generate go run github.com/emersion/go-varlink/cmd/varlinkgen -i varlink/calcApi/org.example.calc.varlink
//go:generate go run github.com/emersion/go-varlink/cmd/varlinkgen -i varlink/stringApi/org.example.string.varlink
