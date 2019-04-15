package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"
)

func newDeclComparer(fset *token.FileSet, path string, summary *packageSummary) *declComparer {
	return &declComparer{
		fset: fset,
		path: path,
		head: summary,
	}
}

// declComparer is an ast.Visitor which ensures that all encountered
// declarations have been unmodified in the provided summary.
type declComparer struct {
	fset   *token.FileSet
	path   string // relative folder
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
		c.logf("\n• Removed in working tree:")
		c.logPosition(base, true)
		c.logf("      type %s", base.Name.Name)
		return
	}
	if x, ok := base.Type.(*ast.StructType); ok {
		// compare structs, allow adding new fields
		y, ok := head.Type.(*ast.StructType)
		if !ok {
			c.logf("\n• Type changed to %s:", describeType(head.Type))
			c.logPosition(base, true)
			c.logf("      type %s", base.Name.Name)
		}
		c.compareStructs(base.Name.Name, x, y)
		return
	}
	if a, b := describeType(base.Type), describeType(head.Type); a != b {
		c.logf("\n• Type changed from %q to %q:", a, b)
		c.logPosition(base, true)
		c.logf("      type %s", base.Name.Name)
	}
}

func (c *declComparer) compareStructs(structName string, base, head *ast.StructType) {
	findType := func(name string, typ ast.Expr) {
		for _, f := range head.Fields.List {
			for _, n := range f.Names {
				if n.Name == name {
					if a, b := describeType(f.Type), describeType(typ); a != b {
						c.logf("\n• Struct field %q type changed from %q to %q:", name, b, a)
						c.logPosition(base, true)
						c.logf("      struct %s", structName)
					}
					return
				}
			}
		}
		c.logf("\n• Removed struct field %q:", name)
		c.logPosition(base, true)
		c.logf("      struct %s", structName)
		c.logPosition(head, false)
		c.logf("      struct %s", structName)
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
			c.logf("\n• Value removed in working tree:")
			c.logPosition(base, true)
			c.logf("      %s", printValue(base))
			return
		}
		if a, b := describeType(head.Type), describeType(base.Type); a != b {
			c.logf("\n• Value type changed from %q to %q in working tree:", b, a)
			c.logPosition(base, true)
			c.logf("      %s", name.Name)
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
		c.logf("\n• Func removed:")
		c.logPosition(base, true)
		c.logf("      %s", printFunc(base.Recv, base.Name, base.Type))
		return
	}
	headArgs := head.Type.Params.List
	baseArgs := base.Type.Params.List
	if diff := len(headArgs) - len(baseArgs); diff != 0 {
		// if there's only one new argument in base, and that argument
		// is variadic, then this isn't a breaking change
		if diff != 1 || !strings.HasPrefix(describeType(headArgs[len(headArgs)-1].Type), "...") {
			c.logFuncChange(base, head, "Change in argument count")
			return
		}
	}
	for i, arg := range baseArgs {
		if a, b := describeType(arg.Type), describeType(headArgs[i].Type); a != b {
			c.logFuncChange(base, head, fmt.Sprintf("Argument (%d) changed from %q to %q", i, a, b))
			return
		}
	}
	baseResults := base.Type.Results
	headResults := head.Type.Results
	if baseResults == nil && headResults != nil {
		c.logFuncChange(base, head, "Return values were added")
		return
	}
	if baseResults != nil && headResults == nil {
		c.logFuncChange(base, head, "Return values were removed")
		return
	}
	if baseResults == nil && headResults == nil {
		// OK
		return
	}
	if len(baseResults.List) != len(headResults.List) {
		c.logFuncChange(base, head, "Change in return value count")
		return
	}
	for i, arg := range baseResults.List {
		if a, b := describeType(arg.Type), describeType(headResults.List[i].Type); a != b {
			c.logFuncChange(base, head, fmt.Sprintf("Return value (%d) changed from %q to %q", i, a, b))
			return
		}
	}
}

func (c *declComparer) logPosition(node ast.Node, base bool) {
	fset := c.fset
	path := c.path
	gitr := "@" + *baseRef
	if !base {
		fset = c.head.fset
		path = c.head.path
		gitr = ""
	}
	pos := fset.Position(node.Pos())
	c.logf("  - %s:%d%s:", strings.TrimPrefix(pos.Filename, path+"/"), pos.Line, gitr)
}

func (c *declComparer) logFuncChange(base, head *ast.FuncDecl, reason string) {
	c.logf("\n• %s:", reason)
	c.logPosition(base, true)
	c.logf("      %s", printFunc(base.Recv, base.Name, base.Type))
	c.logPosition(head, false)
	c.logf("      %s", printFunc(head.Recv, head.Name, head.Type))
}

func (c *declComparer) logf(format string, args ...interface{}) {
	if c.report.Len() == 0 {
		c.report.WriteString(c.head.path)
		c.report.WriteByte(':')
		c.report.WriteByte('\n')
	}
	c.report.WriteString("  ")
	c.report.WriteString(fmt.Sprintf(format, args...))
	c.report.WriteByte('\n')
}
