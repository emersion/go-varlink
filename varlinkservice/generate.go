//go:build generate

package varlinkservice

import (
	_ "git.sr.ht/~emersion/go-varlink/cmd/varlinkgen"
)

//go:generate go run git.sr.ht/~emersion/go-varlink/cmd/varlinkgen -i org.varlink.service.varlink
