package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"testing"
)

// Portable source-level regression guard for Windows-only UI code. It runs
// on Linux CI even though the Win32 GUI itself cannot execute there.
func TestMainWindowDropdownsUseNativeWindowsCombobox(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "main.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	funcs := make(map[string]*ast.FuncDecl)
	for _, d := range file.Decls {
		if fn, ok := d.(*ast.FuncDecl); ok {
			funcs[fn.Name.Name] = fn
		}
	}
	for _, name := range []string{"createCombo", "comboReset", "comboAdd", "onCommand", "comboSetIndex"} {
		if funcs[name] == nil {
			t.Fatalf("missing native dropdown function %s", name)
		}
	}
	hasCall := func(fn *ast.FuncDecl, name string) bool {
		found := false
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			c, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			switch f := c.Fun.(type) {
			case *ast.Ident:
				if f.Name == name {
					found = true
				}
			case *ast.SelectorExpr:
				if f.Sel.Name == name {
					found = true
				}
			}
			return true
		})
		return found
	}
	create := funcs["createCombo"]
	if !hasCall(create, "createControl") {
		t.Fatal("native ComboBox control not created")
	}
	if hasCall(create, "createButton") {
		t.Fatal("custom button dropdown reintroduced")
	}
	foundClass := false
	ast.Inspect(create.Body, func(n ast.Node) bool {
		lit, ok := n.(*ast.BasicLit)
		if ok && lit.Kind == token.STRING {
			if name, err := strconv.Unquote(lit.Value); err == nil && name == "COMBOBOX" {
				foundClass = true
			}
		}
		return true
	})
	if !foundClass {
		t.Fatal("main-window ComboBox class missing")
	}
	for _, pair := range [][2]string{{"comboReset", "CB_RESETCONTENT"}, {"comboAdd", "CB_ADDSTRING"}, {"comboSetIndex", "CB_SETCURSEL"}, {"onCommand", "CB_GETCURSEL"}} {
		found := false
		ast.Inspect(funcs[pair[0]].Body, func(n ast.Node) bool {
			if id, ok := n.(*ast.Ident); ok && id.Name == pair[1] {
				found = true
			}
			return true
		})
		if !found {
			t.Errorf("%s does not use %s", pair[0], pair[1])
		}
	}
}
