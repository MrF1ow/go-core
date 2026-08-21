package coreapp

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"testing"
)

func TestOperatorJSONRoutesRequirePermissionOnEachRegistration(t *testing.T) {
	source, err := os.ReadFile("app.go")
	if err != nil {
		t.Fatal(err)
	}

	unguarded, err := unguardedOperatorRouteLines(source)
	if err != nil {
		t.Fatal(err)
	}
	if len(unguarded) != 0 {
		t.Fatalf("operator routes without requireOp on the same statement: %v", unguarded)
	}
}

func TestUnguardedOperatorRouteLinesDetectsMissingRequireOp(t *testing.T) {
	source := []byte(`package fixture
func register() {
	adminRoutes.POST("/tenants", a.adminHandler.CreateTenant)
}`)

	unguarded, err := unguardedOperatorRouteLines(source)
	if err != nil {
		t.Fatal(err)
	}
	if len(unguarded) != 1 || unguarded[0] != 3 {
		t.Fatalf("unguarded lines = %v, want [3]", unguarded)
	}
}

func unguardedOperatorRouteLines(source []byte) ([]int, error) {
	files := token.NewFileSet()
	file, err := parser.ParseFile(files, "app.go", source, 0)
	if err != nil {
		return nil, fmt.Errorf("parse route source: %w", err)
	}

	var unguarded []int
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || !isOperatorRouteRegistration(selector) || len(call.Args) < 2 {
			return true
		}
		for _, arg := range call.Args[1:] {
			middlewareCall, ok := arg.(*ast.CallExpr)
			if !ok {
				continue
			}
			name, ok := middlewareCall.Fun.(*ast.Ident)
			if ok && name.Name == "requireOp" {
				return true
			}
		}
		unguarded = append(unguarded, files.Position(call.Pos()).Line)
		return true
	})
	return unguarded, nil
}

func isOperatorRouteRegistration(selector *ast.SelectorExpr) bool {
	group, ok := selector.X.(*ast.Ident)
	if !ok {
		return false
	}
	switch group.Name {
	case "adminRoutes", "metricsGroup", "adminOIDC":
	default:
		return false
	}
	switch selector.Sel.Name {
	case "GET", "POST", "PUT", "PATCH", "DELETE":
		return true
	default:
		return false
	}
}
