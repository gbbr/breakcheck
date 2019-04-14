package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"io"
	"strings"
)

func printFuncDecl(v *ast.FuncDecl) { fmt.Println(printFunc(v.Recv, v.Name, v.Type)) }

func printFunc(recv *ast.FieldList, name *ast.Ident, typ *ast.FuncType) string {
	var s strings.Builder
	s.WriteString("func ")
	if name != nil {
		if recv := fieldListType(recv); recv != "" {
			s.WriteByte('(')
			s.WriteString(recv)
			s.WriteByte(')')
			s.WriteByte(' ')
		}
		s.WriteString(name.Name)
	}
	s.WriteByte('(')
	writeArgs(&s, describeFieldList(typ.Params))
	s.WriteByte(')')
	if typ.Results != nil && len(typ.Results.List) > 0 {
		s.WriteByte(' ')
		s.WriteByte('(')
		writeArgs(&s, describeFieldList(typ.Results))
		s.WriteByte(')')
	}
	return s.String()
}

func printValueSpec(v *ast.ValueSpec) { fmt.Println(printValue(v)) }

func printValue(v *ast.ValueSpec) string {
	var s strings.Builder
	t := describeType(v.Type)
	// TODO(gbbr): doesn't work for untyped values (var A = ...)
	for _, name := range v.Names {
		if !ast.IsExported(name.Name) {
			continue
		}
		if name.Obj != nil {
			switch name.Obj.Kind {
			case ast.Con:
				s.WriteString("const ")
			case ast.Var:
				s.WriteString("var ")
			}
		}
		s.WriteString(name.Name)
		s.WriteByte(' ')
		s.WriteString(t)
	}
	if s.Len() > 0 {
		return s.String()
	}
	return ""
}

func printTypeSpec(v *ast.TypeSpec) { fmt.Println(printType(v)) }

func printType(v *ast.TypeSpec) string {
	return fmt.Sprintf("type %s %s", v.Name.Name, describeType(v.Type))
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

func fieldListType(list *ast.FieldList) string {
	types := describeFieldList(list)
	if len(types) == 0 {
		return ""
	}
	return types[0]
}

func describeFieldList(list *ast.FieldList) []string {
	if list == nil {
		return nil
	}
	names := make([]string, 0, len(list.List))
	for _, field := range list.List {
		if field == nil {
			continue
		}
		names = append(names, describeType(field.Type))
	}
	return names
}

func describeType(t ast.Expr) string {
	var name string
	ast.Inspect(t, func(n ast.Node) bool {
		if n == nil {
			return false
		}
		switch v := n.(type) {
		case *ast.Ident:
			name = v.Name
			return false

		case *ast.ChanType:
			name = "chan"
			if v.Arrow != token.NoPos {
				switch v.Dir {
				case ast.SEND:
					name = "chan<-"
				case ast.RECV:
					name = "<-chan"
				}
			}
			return false

		case *ast.MapType:
			name = "map[" + describeType(v.Key) + "]" + describeType(v.Value)
			return false

		case *ast.FuncType:
			var s1 strings.Builder
			writeArgs(&s1, describeFieldList(v.Params))
			if v.Results != nil && len(v.Results.List) > 0 {
				var s2 strings.Builder
				writeArgs(&s2, describeFieldList(v.Results))
				name = fmt.Sprintf("F(%v) (%v)", s1.String(), s2.String())
				return false
			}
			name = fmt.Sprintf("func(%v)", s1.String())
			return false

		case *ast.InterfaceType:
			name = describeInterfaceType(v)
			return false

		case *ast.StructType:
			name = describeStructType(v)
			return false

		case *ast.ArrayType:
			if v.Len == nil {
				name = "[]" + describeType(v.Elt)
				return false
			}
			switch x := v.Len.(type) {
			case *ast.Ellipsis:
				name = "[...]" + describeType(v.Elt)
				return false
			case *ast.BasicLit:
				name = "[" + x.Value + "]" + describeType(v.Elt)
				return false
			case *ast.Ident:
				name = "[" + x.Name + "]" + describeType(v.Elt)
				return false
			default:
				// TODO: this can't stay
				panic(fmt.Sprintf("unexpected array type %T: [%#v]%s\n", v.Len, v.Len, describeType(v.Elt)))
			}

		case *ast.Ellipsis:
			name = "..." + describeType(v.Elt)
			return false

		case *ast.SelectorExpr:
			name = describeType(v.X) + "." + describeType(v.Sel)
			return false

		case *ast.StarExpr:
			name = "*" + describeType(v.X)
			return false

		default:
			panic(fmt.Sprintf("unhandled type: %#v", t))
		}
		return true
	})
	return name
}

func describeStructType(v *ast.StructType) string {
	if v.Fields == nil {
		return "struct{}"
	}
	var s strings.Builder
	s.WriteString("struct{")
	for i, field := range v.Fields.List {
		if len(field.Names) == 0 {
			// should not be possible
			continue
		}
		var j int
		for _, name := range field.Names {
			if !ast.IsExported(name.Name) {
				continue
			}
			if j == 0 {
				if i == 0 {
					s.WriteByte('\n')
				}
				s.WriteByte('\t')
			}
			if j > 0 {
				s.WriteString(", ")
			}
			s.WriteString(name.Name)
			j++
		}
		if j == 0 {
			continue
		}
		s.WriteByte(' ')
		s.WriteString(describeType(field.Type))
		s.WriteByte('\n')
	}
	s.WriteByte('}')
	return s.String()
}

func describeInterfaceType(v *ast.InterfaceType) string {
	if v.Methods == nil {
		return "interface{}"
	}
	var s strings.Builder
	s.WriteString("interface{")
	for i, field := range v.Methods.List {
		if i == 0 {
			s.WriteByte('\n')
		}
		ft, ok := field.Type.(*ast.FuncType)
		if !ok {
			// should not be possible
			continue
		}
		s.WriteByte('\t')
		s.WriteString(printFunc(nil, field.Names[0], ft))
		s.WriteByte('\n')
	}
	s.WriteByte('}')
	return s.String()
}
