package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

// This test parses source; it never opens or queries a running process.
func TestProcessBackendContainsOnlyReadAndQueryAPIs(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "process_windows.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	allowed := map[string]bool{"OpenProcess": true, "CloseHandle": true, "ReadProcessMemory": true, "CreateToolhelp32Snapshot": true,
		"Process32First": true, "Process32Next": true, "Module32First": true, "Module32Next": true, "QueryFullProcessImageName": true, "UTF16ToString": true}
	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		pkg, ok := sel.X.(*ast.Ident)
		if !ok || pkg.Name != "windows" {
			return true
		}
		if !allowed[sel.Sel.Name] {
			t.Errorf("non-read/query API %s", sel.Sel.Name)
		}
		if sel.Sel.Name == "OpenProcess" {
			ast.Inspect(call.Args[0], func(n ast.Node) bool {
				if s, ok := n.(*ast.SelectorExpr); ok && strings.HasPrefix(s.Sel.Name, "PROCESS_") && s.Sel.Name != "PROCESS_VM_READ" && s.Sel.Name != "PROCESS_QUERY_INFORMATION" && s.Sel.Name != "PROCESS_QUERY_LIMITED_INFORMATION" {
					t.Errorf("excess rights %s", s.Sel.Name)
				}
				return true
			})
		}
		return true
	})
}
