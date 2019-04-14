package main

import (
	"go/ast"
	"strings"
)

type packageSummary struct {
	path  string                    // relative folder
	funcs map[string]*ast.FuncDecl  // maps (recv.)name to func
	types map[string]*ast.TypeSpec  // maps type name to spec
	value map[string]*ast.ValueSpec // maps value name to spec
	// TODO: add package name, ensure it wasn't changed
}

func newPackageSummary(path string) *packageSummary {
	return &packageSummary{
		path:  path,
		funcs: make(map[string]*ast.FuncDecl),
		value: make(map[string]*ast.ValueSpec),
		types: make(map[string]*ast.TypeSpec),
	}
}

func (ps *packageSummary) Visit(node ast.Node) ast.Visitor {
	if node == nil {
		return nil
	}
	switch v := node.(type) {
	case *ast.FuncDecl:
		if v.Name == nil {
			return nil
		}
		name, ok := funcName(v)
		if !ok {
			return nil
		}
		ps.funcs[name] = v
		return nil

	case *ast.TypeSpec:
		ps.types[v.Name.Name] = v
		return nil

	case *ast.ValueSpec:
		for _, name := range v.Names {
			ps.value[name.Name] = v
		}
		return nil

	default:
		return ps
	}
}

// funcName returns the full func name (recv.name) and true if the receiver
// is exported.
func funcName(v *ast.FuncDecl) (string, bool) {
	name := v.Name.Name
	if recv := fieldListType(v.Recv); recv != "" {
		name = recv + "." + v.Name.Name
		if !ast.IsExported(strings.TrimPrefix(recv, "(*")) {
			// an exported method un an unexported receiver, skip
			return name, false
		}
	}
	return name, true
}
