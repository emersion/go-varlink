package main

import (
	"flag"
	"log"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("usage: certification [ client | server ] [ -protocol PROTOCOL ] [ -socket SOCKET]")
	}

	mode := os.Args[1]
	cmd := flag.NewFlagSet(mode, flag.ExitOnError)
	protocol := cmd.String("protocol", "tcp", "Protocol (tcp, unix, ...)")
	socket := cmd.String("socket", "127.0.0.1:12345", "Socket address")
	cmd.Parse(os.Args[2:])

	switch mode {
	case "client":
		client(*protocol, *socket)
	default:
		log.Fatal("usage: certification [ client | server ] FLAGS")
	}
}
