// Package varlinkdef implements the Varlink interface definition format.
//
// See: https://varlink.org/Interface-Definition
package varlinkdef

import (
	"fmt"
)

type Interface struct {
	Name    string
	Types   map[string]Type
	Methods map[string]Method
	Errors  map[string]Struct
}

type Method struct {
	In, Out Struct
}

type Enum []string

type Struct map[string]Type

type Kind int

const (
	KindStruct Kind = iota + 1
	KindEnum
	KindName
	KindBool
	KindInt
	KindFloat
	KindString
	KindObject
	KindArray
	KindMap
)

func (kind Kind) String() string {
	switch kind {
	case KindStruct:
		return "struct"
	case KindEnum:
		return "enum"
	case KindName:
		return "name"
	case KindBool:
		return "bool"
	case KindInt:
		return "int"
	case KindFloat:
		return "float"
	case KindString:
		return "string"
	case KindObject:
		return "object"
	case KindArray:
		return "array"
	case KindMap:
		return "map"
	default:
		panic(fmt.Errorf("invalid kind %v", int(kind)))
	}
}

type Type struct {
	Kind     Kind
	Nullable bool
	Inner    *Type  // for KindArray and KindMap
	Name     string // for KindName
	Struct   Struct // for KindStruct
	Enum     Enum   // for KindEnum
}

var (
	TypeBool   = Type{Kind: KindBool}
	TypeInt    = Type{Kind: KindInt}
	TypeFloat  = Type{Kind: KindFloat}
	TypeString = Type{Kind: KindString}
	TypeObject = Type{Kind: KindObject}
)
