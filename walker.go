package main

import (
	"fmt"
	"go/ast"
	"io"
	"strings"
)

type typeData struct {
	name    string
	members []string
}

type publicAPI struct {
	types []*typeData
	funcs []*funcData
}

func newPublicAPI() *publicAPI {
	return &publicAPI{}
}

func (c *publicAPI) Visit(node ast.Node) ast.Visitor {
	if node == nil {
		return nil
	}
	switch v := node.(type) {
	case *ast.FuncDecl:
		// A FuncDecl node represents a function declaration.
		if v.Name == nil || !ast.IsExported(v.Name.Name) {
			return nil
		}
		fd := newFuncData(v)
		if recv := fd.recv; len(recv) > 0 {
			if !ast.IsExported(strings.TrimPrefix(recv, "*")) {
				// an exported method un an unexported struct, skip
				return nil
			}
		}
		fmt.Printf("> %s\n", fd)
		c.funcs = append(c.funcs, fd)
		return nil
	case *ast.TypeSpec:
		// A TypeSpec node represents a type declaration (TypeSpec production).
		if !ast.IsExported(v.Name.Name) {
			return nil
		}
		fmt.Printf("> T %s %s\n", v.Name.Name, describeType(v.Type))
		return nil
	case *ast.ValueSpec:
		// A ValueSpec node represents a constant or variable declaration
		// (ConstSpec or VarSpec production).
		if len(v.Names) == 0 {
			return nil
		}
		desc, ok := describeValue(v)
		if ok {
			fmt.Printf("> V %s\n", desc)
		}
		return nil
	default:
		return c
	}
}

type funcData struct {
	name string
	recv string
	args []string
	rets []string
}

func newFuncData(v *ast.FuncDecl) *funcData {
	d := &funcData{}
	if v.Name != nil {
		d.name = v.Name.Name
	}
	d.recv = fieldListType(v.Recv)
	if v.Type != nil {
		d.args = describeFieldList(v.Type.Params)
		d.rets = describeFieldList(v.Type.Results)
	}
	return d
}

func (fd *funcData) String() string {
	var s strings.Builder
	s.WriteString("F ")
	if fd.recv != "" {
		s.WriteString("(")
		s.WriteString(fd.recv)
		s.WriteString(")")
		s.WriteByte(' ')
	}
	s.WriteString(fd.name)
	s.WriteByte('(')
	writeArgs(&s, fd.args)
	s.WriteString(") ")
	if len(fd.rets) > 0 {
		s.WriteByte('(')
		writeArgs(&s, fd.rets)
		s.WriteByte(')')
	}
	return s.String()
}

// writeArgs pretty-prints a list of arguments (e.g. ["1", "2", "3"] => "1, 2, 3")
func writeArgs(s io.StringWriter, args []string) {
	for i, arg := range args {
		s.WriteString(arg)
		if i < len(args)-1 {
			s.WriteString(", ")
		}
	}
}
