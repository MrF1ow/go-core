# GUI shell signatures

Companion to [overview.md](overview.md). `not implemented` bodies stay in the implementation PR.

## Public surface

```go
// internal/coreapp/app.go
func requireGUI(resource, action string) gin.HandlerFunc

// internal/middleware
func RequireGUIPermission(resource, action string) gin.HandlerFunc
func AbortGUIForbidden(c *gin.Context)
func AbortGUIInternal(c *gin.Context)

// internal/admin
func (h *GUIHandler) page(c *gin.Context) web.TemplateData

// internal/operator
func AssignableSystemRoles(p Principal) []SystemRole
func ParseAssignedAdminRole(p Principal, postedRoleID, keyType string) (*uuid.UUID, error)
var ErrIAMAssignmentDenied error
```

`requireGUI` exists so the inventory AST has a stable Ident, same as `requireOp`.

## TemplateData

```go
type TemplateData struct {
    // existing fields...
    NavGroups []NavGroup
    can       func(resource, action string) bool
}

func (td TemplateData) Can(resource, action string) bool
```

Nil `can` returns false.

## Nav

```go
type NavSpec struct {
    Heading, Page, Path, Icon, Label, Resource, Action string
}

type NavGroup struct {
    Heading string
    Items   []NavItem
}

func buildNav(basePath string, can func(string, string) bool) []NavGroup
```

`admin_iam` has no nav row in this slice.

## Session-only allowlist

```go
Exact:  []string{"/logout"}
Prefix: []string{"/my-account"}
```

Inventory scans `guiAuth` and identifiers assigned from `guiAuth.Group(...)`.

## Deny header

```
X-GUI-Forbidden: 1
```

CSRF and settings env-lock must not send it.

## Templates

- `web/templates/pages/forbidden.tmpl` for typed URL and `#page-content`
- `web/templates/partials/forbidden_fragment.tmpl` for other HTMX targets

Do not name either `"error"`. That template does not exist.
