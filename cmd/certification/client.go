package main

import (
	"io"
	"log"
	"net"

	"github.com/emersion/go-varlink"
	"github.com/emersion/go-varlink/internal/certification"
)

func client(protocol, socket string) {
	log.Printf("Connecting to %s://%s\n", protocol, socket)
	conn, err := net.Dial(protocol, socket)
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	c := certification.Client{Client: varlink.NewClient(conn)}

	log.Println("Start(): nil -> str")
	startOut, err := c.Start(nil)
	if err != nil {
		panic(err)
	}
	clientID := startOut.ClientId
	log.Println("Start response:", startOut)

	log.Println("Test01: string -> bool")
	test01Out, err := c.Test01(&certification.Test01In{ClientId: clientID})
	if err != nil {
		panic(err)
	}
	log.Println("Test01 response:", test01Out)

	log.Println("Test02: bool -> int")
	test02Out, err := c.Test02(&certification.Test02In{
		ClientId: clientID,
		Bool:     test01Out.Bool,
	})
	if err != nil {
		panic(err)
	}
	log.Println("Test02 response:", test02Out)

	log.Println("Test03: int -> float")
	test03Out, err := c.Test03(&certification.Test03In{
		ClientId: clientID,
		Int:      test02Out.Int,
	})
	if err != nil {
		panic(err)
	}
	log.Println("Test03 response:", test03Out)

	log.Println("Test04: float -> string")
	test04Out, err := c.Test04(&certification.Test04In{
		ClientId: clientID,
		Float:    test03Out.Float,
	})
	if err != nil {
		panic(err)
	}
	log.Println("Test04 response:", test04Out)

	log.Println("Test05: string -> multiple values")
	test05Out, err := c.Test05(&certification.Test05In{
		ClientId: clientID,
		String:   test04Out.String,
	})
	if err != nil {
		panic(err)
	}
	log.Println("Test05 response:", test05Out)

	log.Println("Test06: multiple values -> struct")
	test06Out, err := c.Test06(&certification.Test06In{
		ClientId: clientID,
		Bool:     test05Out.Bool,
		Int:      test05Out.Int,
		Float:    test05Out.Float,
		String:   test05Out.String,
	})
	if err != nil {
		panic(err)
	}
	log.Println("Test06 response:", test06Out)

	log.Println("Test07: struct -> map")
	test07Out, err := c.Test07(&certification.Test07In{
		ClientId: clientID,
		Struct:   test06Out.Struct,
	})
	if err != nil {
		panic(err)
	}
	log.Println("Test07 response:", test07Out)

	log.Println("Test08: map -> set")
	test08Out, err := c.Test08(&certification.Test08In{
		ClientId: clientID,
		Map:      test07Out.Map,
	})
	if err != nil {
		panic(err)
	}
	log.Println("Test08 response:", test08Out)

	log.Println("Test09: set -> MyType")
	test09Out, err := c.Test09(&certification.Test09In{
		ClientId: clientID,
		Set:      test08Out.Set,
	})
	if err != nil {
		panic(err)
	}
	log.Println("Test09 response:", test09Out)

	log.Println("Test10: MyType -> streaming string replies")
	// FIXME varlinkgen should generate a streaming API for c.Test10
	call, err := c.Client.DoMore("org.varlink.certification.Test10", &certification.Test10In{
		ClientId: clientID,
		Mytype:   test09Out.Mytype,
	})
	if err != nil {
		panic(err)
	}

	var test10Strings []string
	for {
		test10Out := new(certification.Test10Out)
		err = call.Next(test10Out)
		if err == io.EOF {
			break
		}
		if err != nil {
			panic(err)
		}
		log.Println("Test10 response:", test10Out)
		test10Strings = append(test10Strings, test10Out.String)
	}

	log.Println("Test11: oneway call")
	// FIXME varlinkgen should generate a oneway API for c.Test11
	err = c.Client.DoOneway("org.varlink.certification.Test11", &certification.Test11In{
		ClientId:        clientID,
		LastMoreReplies: test10Strings,
	})
	if err != nil {
		panic(err)
	}
	log.Println("Test11 completed")

	log.Println("End")
	endOut, err := c.End(&certification.EndIn{ClientId: clientID})
	if err != nil {
		panic(err)
	}

	if endOut.AllOk {
		log.Println("Client certification passed!")
		return
	}
	panic("certification failed")
}
