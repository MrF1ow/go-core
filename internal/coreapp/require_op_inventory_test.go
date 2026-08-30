package coreapp

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strconv"
	"strings"
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

func TestGUIApiKeyDeleteRoutesRemoved(t *testing.T) {
	source, err := os.ReadFile("app.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)
	for _, needle := range []string{
		`"/api-keys/:id/delete"`,
		`DELETE("/api-keys/:id"`,
	} {
		if strings.Contains(text, needle) {
			t.Fatalf("app.go still registers %s", needle)
		}
	}
	for _, needle := range []string{
		`GET("/api-keys/:id/revoke"`,
		`PUT("/api-keys/:id/revoke"`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("app.go missing %s", needle)
		}
	}
}

func TestGUIRoutesRequirePermissionOnEachRegistration(t *testing.T) {
	source, err := os.ReadFile("app.go")
	if err != nil {
		t.Fatal(err)
	}

	unguarded, err := unguardedGUIRouteLines(source)
	if err != nil {
		t.Fatal(err)
	}
	if len(unguarded) != 0 {
		t.Fatalf("guiAuth routes without requireGUI on the same statement: %v", unguarded)
	}
}

func TestUnguardedGUIRouteLinesDetectsMissingRequireGUI(t *testing.T) {
	source := []byte(`package fixture
func register() {
	guiAuth.GET("/tenants", a.guiHandler.TenantPage)
}`)

	unguarded, err := unguardedGUIRouteLines(source)
	if err != nil {
		t.Fatal(err)
	}
	if len(unguarded) != 1 || unguarded[0] != 3 {
		t.Fatalf("unguarded lines = %v, want [3]", unguarded)
	}
}

func TestUnguardedGUIRouteLinesAllowsLogoutAndMyAccount(t *testing.T) {
	source := []byte(`package fixture
func register() {
	guiAuth.GET("/logout", a.guiHandler.Logout)
	guiAuth.GET("/my-account/2fa/status", a.guiHandler.MyAccount2FAStatus)
}`)

	unguarded, err := unguardedGUIRouteLines(source)
	if err != nil {
		t.Fatal(err)
	}
	if len(unguarded) != 0 {
		t.Fatalf("unguarded lines = %v, want none", unguarded)
	}
}

func TestUnguardedGUIRouteLinesDetectsNestedSessions(t *testing.T) {
	source := []byte(`package fixture
func register() {
	guiAuth.GET("/users/:id/sessions", a.guiHandler.UserSessions)
}`)

	unguarded, err := unguardedGUIRouteLines(source)
	if err != nil {
		t.Fatal(err)
	}
	if len(unguarded) != 1 || unguarded[0] != 3 {
		t.Fatalf("unguarded lines = %v, want [3]", unguarded)
	}
}

func TestUnguardedGUIRouteLinesScansGroupDerivatives(t *testing.T) {
	source := []byte(`package fixture
func register() {
	inner := guiAuth.Group("/nested")
	inner.GET("/tenants", a.guiHandler.TenantPage)
}`)

	unguarded, err := unguardedGUIRouteLines(source)
	if err != nil {
		t.Fatal(err)
	}
	if len(unguarded) != 1 || unguarded[0] != 4 {
		t.Fatalf("unguarded lines = %v, want [4]", unguarded)
	}
}

func unguardedGUIRouteLines(source []byte) ([]int, error) {
	files := token.NewFileSet()
	file, err := parser.ParseFile(files, "app.go", source, 0)
	if err != nil {
		return nil, fmt.Errorf("parse route source: %w", err)
	}

	groups := guiAuthGroupNames(file)
	var unguarded []int
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || !isHTTPMethodSelector(selector) || len(call.Args) < 2 {
			return true
		}
		recv, ok := selector.X.(*ast.Ident)
		if !ok || !groups[recv.Name] {
			return true
		}
		if guiSessionOnlyPath(call.Args[0]) {
			return true
		}
		for _, arg := range call.Args[1:] {
			middlewareCall, ok := arg.(*ast.CallExpr)
			if !ok {
				continue
			}
			name, ok := middlewareCall.Fun.(*ast.Ident)
			if ok && name.Name == "requireGUI" {
				return true
			}
		}
		unguarded = append(unguarded, files.Position(call.Pos()).Line)
		return true
	})
	return unguarded, nil
}

func guiAuthGroupNames(file *ast.File) map[string]bool {
	names := map[string]bool{"guiAuth": true}
	changed := true
	for changed {
		changed = false
		ast.Inspect(file, func(node ast.Node) bool {
			assign, ok := node.(*ast.AssignStmt)
			if !ok || len(assign.Lhs) != 1 || len(assign.Rhs) != 1 {
				return true
			}
			ident, ok := assign.Lhs[0].(*ast.Ident)
			if !ok || names[ident.Name] {
				return true
			}
			call, ok := assign.Rhs[0].(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || sel.Sel.Name != "Group" {
				return true
			}
			recv, ok := sel.X.(*ast.Ident)
			if !ok || !names[recv.Name] {
				return true
			}
			names[ident.Name] = true
			changed = true
			return true
		})
	}
	return names
}

func isHTTPMethodSelector(selector *ast.SelectorExpr) bool {
	switch selector.Sel.Name {
	case "GET", "POST", "PUT", "PATCH", "DELETE":
		return true
	default:
		return false
	}
}

func guiSessionOnlyPath(arg ast.Expr) bool {
	lit, ok := arg.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return false
	}
	path, err := strconv.Unquote(lit.Value)
	if err != nil {
		return false
	}
	if path == "/logout" {
		return true
	}
	return path == "/my-account" || strings.HasPrefix(path, "/my-account/")
}
