package varlinkdef

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

func Read(r io.Reader) (*Interface, error) {
	dec := decoder{br: bufio.NewReader(r)}
	return dec.readInterface()
}

type decoder struct {
	br *bufio.Reader
}

func (dec *decoder) skipComment() error {
	for {
		ch, err := dec.br.ReadByte()
		if err != nil {
			return err
		}
		if ch == '\n' {
			return nil
		}
	}
}

func (dec *decoder) skipWhitespace() error {
	for {
		ch, err := dec.br.ReadByte()
		if err == io.EOF {
			return nil
		} else if err != nil {
			return err
		}

		switch ch {
		case ' ', '\t', '\r', '\n':
			// skip
		case '#':
			if err := dec.skipComment(); err != nil {
				return err
			}
		default:
			dec.br.UnreadByte()
			return nil
		}
	}
}

func (dec *decoder) readToken() (string, error) {
	if err := dec.skipWhitespace(); err != nil {
		return "", err
	}

	var sb strings.Builder
	for {
		ch, err := dec.br.ReadByte()
		if err == io.EOF && sb.Len() > 0 {
			return sb.String(), nil
		} else if err != nil {
			return "", err
		}
		switch ch {
		case '?', '(', ')', ',', ':':
			if sb.Len() > 0 {
				dec.br.UnreadByte()
				return sb.String(), nil
			} else {
				return string(ch), nil
			}
		case ']', '>':
			sb.WriteByte(ch)
			return sb.String(), nil
		case ' ', '\t', '\r', '\n', '#':
			dec.br.UnreadByte()
			return sb.String(), nil
		default:
			sb.WriteByte(ch)
		}
	}
}

func (dec *decoder) expectToken(token string) error {
	got, err := dec.readToken()
	if err != nil {
		return fmt.Errorf("in %q: %v", token, err)
	} else if got != token {
		return fmt.Errorf("expected %q, got %q", token, got)
	}
	return nil
}

func (dec *decoder) readInterfaceName() (string, error) {
	name, err := dec.readToken()
	if err != nil {
		return "", fmt.Errorf("in interface name: %v", err)
	} else if !isInterfaceName(name) {
		return "", fmt.Errorf("invalid interface name %q", name)
	}
	return name, nil
}

func (dec *decoder) readName() (string, error) {
	name, err := dec.readToken()
	if err != nil {
		return "", fmt.Errorf("in name: %v", err)
	} else if !isName(name) {
		return "", fmt.Errorf("invalid name %q", name)
	}
	return name, nil
}

func (dec *decoder) readStructOrEnum() (*Type, error) {
	if err := dec.expectToken("("); err != nil {
		return nil, err
	}

	var typ Type
loop:
	for {
		token, err := dec.readToken()
		if err != nil {
			return nil, fmt.Errorf("in struct or enum: %v", err)
		} else if token == ")" && typ.Kind == 0 { // empty parentheses
			typ.Kind = KindStruct
			typ.Struct = Struct{}
			break
		} else if !isFieldName(token) {
			return nil, fmt.Errorf(`expected field name, got %q`, token)
		}
		name := token

		sep, err := dec.readToken()
		if err != nil {
			return nil, fmt.Errorf("in struct or enum: %v", err)
		}
		if typ.Kind == 0 {
			switch sep {
			case ",", ")":
				typ.Kind = KindEnum
				typ.Enum = Enum{}
			case ":":
				typ.Kind = KindStruct
				typ.Struct = Struct{}
			default:
				return nil, fmt.Errorf(`expected one of "," or ":", got %q`, sep)
			}
		} else {
			switch typ.Kind {
			case KindEnum:
				if sep != "," && sep != ")" {
					return nil, fmt.Errorf(`expected "," or ")", got %q`, sep)
				}
			case KindStruct:
				if sep != ":" {
					return nil, fmt.Errorf(`expected ":", got %q`, sep)
				}
			}
		}

		switch typ.Kind {
		case KindEnum:
			typ.Enum = append(typ.Enum, name)
			if sep == ")" {
				break loop
			}
		case KindStruct:
			t, err := dec.readType()
			if err != nil {
				return nil, fmt.Errorf("in struct: %v", err)
			}
			typ.Struct[name] = *t

			sep, err := dec.readToken()
			if err != nil {
				return nil, fmt.Errorf("in struct: %v", err)
			}
			switch sep {
			case ")":
				break loop
			case ",":
				// ok
			default:
				return nil, fmt.Errorf(`expected "," or ")", got %q`, sep)
			}
		}
	}

	return &typ, nil
}

func (dec *decoder) readStruct() (Struct, error) {
	typ, err := dec.readStructOrEnum()
	if err != nil {
		return nil, err
	} else if typ.Kind != KindStruct {
		return nil, fmt.Errorf("expected struct, got %v", typ.Kind)
	}
	return typ.Struct, nil
}

func (dec *decoder) readElementType(token string) (*Type, error) {
	if token == "" {
		var err error
		token, err = dec.readToken()
		if err != nil {
			return nil, fmt.Errorf("in element type: %v", err)
		}
	}

	if kind := parseBasicType(token); kind != 0 {
		return &Type{Kind: kind}, nil
	}

	if token == "(" {
		dec.br.UnreadByte()
		return dec.readStructOrEnum()
	}

	if isName(token) {
		return &Type{Kind: KindName, Name: token}, nil
	}

	return nil, fmt.Errorf("expected element type, got %q", token)
}

func (dec *decoder) readType() (*Type, error) {
	token, err := dec.readToken()
	if err != nil {
		return nil, fmt.Errorf("in type: %v", err)
	}

	nullable := token == "?"
	if nullable {
		token, err = dec.readToken()
		if err != nil {
			return nil, fmt.Errorf("in type: %v", err)
		}
	}

	var kind Kind
	switch token {
	case "[]":
		kind = KindArray
	case "[string]":
		kind = KindMap
	default:
		typ, err := dec.readElementType(token)
		if err != nil {
			return nil, err
		}
		typ.Nullable = nullable
		return typ, nil
	}

	inner, err := dec.readType()
	if err != nil {
		return nil, err
	}

	return &Type{Kind: kind, Inner: inner, Nullable: nullable}, nil
}

func (dec *decoder) readMember(iface *Interface) error {
	keyword, err := dec.readToken()
	if err != nil {
		return err
	}

	switch keyword {
	case "type":
		name, err := dec.readName()
		if err != nil {
			return err
		}
		t, err := dec.readStructOrEnum()
		if err != nil {
			return err
		}
		iface.Types[name] = *t
	case "method":
		name, err := dec.readName()
		if err != nil {
			return err
		}
		in, err := dec.readStruct()
		if err != nil {
			return err
		}
		if err := dec.expectToken("->"); err != nil {
			return err
		}
		out, err := dec.readStruct()
		if err != nil {
			return err
		}
		iface.Methods[name] = Method{In: in, Out: out}
	case "error":
		name, err := dec.readName()
		if err != nil {
			return err
		}
		st, err := dec.readStruct()
		if err != nil {
			return err
		}
		iface.Errors[name] = st
	default:
		return fmt.Errorf(`expected one of "type", "method", "error", got %q`, keyword)
	}

	return nil
}

func (dec *decoder) readInterface() (*Interface, error) {
	if err := dec.expectToken("interface"); err != nil {
		return nil, err
	}
	name, err := dec.readInterfaceName()
	if err != nil {
		return nil, err
	}
	iface := &Interface{
		Name:    name,
		Types:   make(map[string]Type),
		Methods: make(map[string]Method),
		Errors:  make(map[string]Struct),
	}
	for {
		if err := dec.readMember(iface); err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}
	}
	return iface, nil
}

func parseBasicType(token string) Kind {
	switch token {
	case "bool":
		return KindBool
	case "int":
		return KindInt
	case "float":
		return KindFloat
	case "string":
		return KindString
	case "object":
		return KindObject
	default:
		return 0
	}
}

func isInterfaceName(s string) bool {
	// TODO: be more strict
	return len(s) > 0 && isAlpha(s[0]) && containsOnly(s[1:], func(ch byte) bool {
		return isAlphaNum(ch) || ch == '-' || ch == '.'
	})
}

func isName(s string) bool {
	return len(s) > 0 && s[0] >= 'A' && s[0] <= 'Z' && containsOnly(s[1:], isAlphaNum)
}

func isFieldName(s string) bool {
	return len(s) > 0 && isAlpha(s[0]) && containsOnly(s[1:], func(ch byte) bool {
		return isAlphaNum(ch) || ch == '_'
	})
}

func containsOnly(s string, f func(byte) bool) bool {
	for i := 0; i < len(s); i++ {
		if !f(s[i]) {
			return false
		}
	}
	return true
}

func isAlphaNum(ch byte) bool {
	return isAlpha(ch) || (ch >= '0' && ch <= '9')
}

func isAlpha(ch byte) bool {
	return (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z')
}
