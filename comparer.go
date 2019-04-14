package main

import (
	"fmt"
	"go/ast"
	"strings"
)

func newDeclComparer(summary *packageSummary) *declComparer {
	return &declComparer{
		head: summary,
	}
}

// declComparer is an ast.Visitor which ensures that all encountered
// declarations have been unmodified in the provided summary.
type declComparer struct {
	head   *packageSummary
	report strings.Builder
}

func (c *declComparer) Visit(node ast.Node) ast.Visitor {
	if node == nil {
		return nil
	}
	switch v := node.(type) {
	case *ast.FuncDecl:
		if v.Name == nil {
			return nil
		}
		c.compareFunc(v)
		return nil

	case *ast.TypeSpec:
		c.compareType(v)
		return nil

	case *ast.ValueSpec:
		c.compareValue(v)
		return nil

	default:
		return c
	}
}

func (c *declComparer) compareType(base *ast.TypeSpec) {
	head, ok := c.head.types[base.Name.Name]
	if !ok {
		c.logf("removed: type %s", base.Name.Name)
		return
	}
	if x, ok := base.Type.(*ast.StructType); ok {
		// compare structs, allow adding new fields
		y, ok := head.Type.(*ast.StructType)
		if !ok {
			c.logf("type %s changed from struct to %s", base.Name.Name, describeType(head.Type))
		}
		c.compareStructs(base.Name.Name, x, y)
		return
	}
	if a, b := describeType(base.Type), describeType(head.Type); a != b {
		c.logf("type changed: %s", base.Name.Name)
		c.logf("\tfrom: %s", a)
		c.logf("\t  to: %s", b)
	}
}

func (c *declComparer) compareStructs(structName string, base, head *ast.StructType) {
	findType := func(name string, typ ast.Expr) {
		for _, f := range head.Fields.List {
			for _, n := range f.Names {
				if n.Name == name {
					if a, b := describeType(f.Type), describeType(typ); a != b {
						c.logf("struct %s field %s changed type from %s to %s", structName, name, b, a)
					}
					return
				}
			}
		}
		c.logf("struct field %s.%s was removed", structName, name)
	}
	for _, field := range base.Fields.List {
		for _, name := range field.Names {
			findType(name.Name, field.Type)
		}
	}
}

func (c *declComparer) compareValue(base *ast.ValueSpec) {
	for _, name := range base.Names {
		head, ok := c.head.value[name.Name]
		if !ok {
			c.logf("removed: %s", printValue(base))
			return
		}
		if a, b := describeType(head.Type), describeType(base.Type); a != b {
			c.logf("type changed for value %s, from %s to %s", name.Name, b, a)
		}
		if base.Type == nil {
			// If the type is nil, this is likely an assignment where the type is inferred
			// at compile time. In that case, a breaking change could be a change in value.
			// TODO
		}
	}
}

func (c *declComparer) compareFunc(base *ast.FuncDecl) {
	name, ok := funcName(base)
	if !ok {
		return
	}
	head, ok := c.head.funcs[name]
	if !ok {
		// func not found
		c.logf("removed: %s", printFunc(base.Recv, base.Name, base.Type))
		return
	}
	headArgs := head.Type.Params.List
	baseArgs := base.Type.Params.List
	if diff := len(headArgs) - len(baseArgs); diff != 0 {
		// if there's only one new argument in base, and that argument
		// is variadic, then this isn't a breaking change
		if diff != 1 || !strings.HasPrefix(describeType(headArgs[len(headArgs)-1].Type), "...") {
			c.logFuncChange(base, head, "change in argument count")
			return
		}
	}
	for i, arg := range baseArgs {
		if a, b := describeType(arg.Type), describeType(headArgs[i].Type); a != b {
			c.logFuncChange(base, head, fmt.Sprintf("argument %d changed from %s to %s", i, a, b))
			return
		}
	}
	baseResults := base.Type.Results
	headResults := head.Type.Results
	if baseResults == nil && headResults != nil {
		c.logFuncChange(base, head, "return values were added")
		return
	}
	if baseResults != nil && headResults == nil {
		c.logFuncChange(base, head, "return values were removed")
		return
	}
	if baseResults == nil && headResults == nil {
		// OK
		return
	}
	if len(baseResults.List) != len(headResults.List) {
		c.logFuncChange(base, head, "change in return value count")
		return
	}
	for i, arg := range baseResults.List {
		if a, b := describeType(arg.Type), describeType(headResults.List[i].Type); a != b {
			c.logFuncChange(base, head, fmt.Sprintf("return value %d changed from %s to %s", i, a, b))
			return
		}
	}
}

func (c *declComparer) logFuncChange(base, head *ast.FuncDecl, reason string) {
	c.logf(" before: %s", printFunc(base.Recv, base.Name, base.Type))
	c.logf("    now: %s", printFunc(head.Recv, head.Name, head.Type))
	c.logf("         (%s)", reason)
}

func (c *declComparer) logf(format string, args ...interface{}) {
	if c.report.Len() == 0 {
		c.report.WriteString(c.head.path)
		c.report.WriteByte(':')
		c.report.WriteByte('\n')
	}
	c.report.WriteByte('\t')
	c.report.WriteString(fmt.Sprintf(format, args...))
	c.report.WriteByte('\n')
}
