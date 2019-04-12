package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"
)

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
			default:
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
		if len(field.Names) == 0 {
			// should not be possible
			continue
		}
		ft, ok := field.Type.(*ast.FuncType)
		if !ok {
			// should not be possible
			continue
		}
		fd := funcData{
			name: field.Names[0].Name,
			args: describeFieldList(ft.Params),
			rets: describeFieldList(ft.Results),
		}
		s.WriteByte('\t')
		s.WriteString(fd.String())
		s.WriteByte('\n')
	}
	s.WriteByte('}')
	return s.String()
}

// returns the value and true if it has at least 1 exported ident.
func describeValue(v *ast.ValueSpec) (string, bool) {
	var s strings.Builder
	var i int
	for _, name := range v.Names {
		if !ast.IsExported(name.Name) {
			continue
		}
		if i > 0 {
			s.WriteString(", ")
		}
		s.WriteString(name.Name)
		i++
	}
	if i == 0 {
		return "", false
	}
	s.WriteByte(' ')
	s.WriteString(describeType(v.Type))
	return s.String(), true
}
