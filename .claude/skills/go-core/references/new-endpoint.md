# Adding New Endpoints & Features to go-core

This skill walks through the full workflow for adding new API functionality to the auth API. The codebase follows a strict Repository-Service-Handler architecture with dependency injection wired in `cmd/api/main.go`.

## The Layer Cake

Every feature follows this exact stack. Build bottom-up:

```
migrations/       → SQL migration (if new table needed)
pkg/models/       → Model struct (if new table needed)
internal/sqlcgen/ → Run `sqlc generate` after adding queries
pkg/dto/          → Request/Response DTOs with validator + Swagger tags
internal/<domain>/repository.go → Data access (pgx pool queries)
internal/<domain>/service.go    → Business logic (interface + impl)
internal/<domain>/handler.go    → HTTP handler with Swagger annotations
cmd/api/main.go                 → Wire everything + register routes
```

## Step 1: Migration & Model (if new table)

Add a migration file to `migrations/` with the SQL:

```sql
CREATE TABLE widgets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    app_id UUID NOT NULL REFERENCES applications(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_widgets_app_id ON widgets(app_id);
```

Add a model struct to `pkg/models/` for reference (used by services, not for DB access):

```go
type Widget struct {
    ID        uuid.UUID `json:"id"`
    AppID     uuid.UUID `json:"app_id"`
    Name      string    `json:"name"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}
```

## Step 2: DTOs

Add to `pkg/dto/`. Always include validator tags and Swagger example tags:

```go
type CreateWidgetRequest struct {
    Name string `json:"name" validate:"required,min=1,max=255" example:"My Widget"`
}

type WidgetResponse struct {
    ID        string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
    Name      string `json:"name" example:"My Widget"`
    CreatedAt string `json:"created_at" example:"2024-01-01T00:00:00Z"`
}
```

## Step 3: Repository

Create `internal/<domain>/repository.go`. Uses pgx pool directly:

```go
type Repository struct {
    pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
    return &Repository{pool: pool}
}

func (r *Repository) Create(ctx context.Context, appID uuid.UUID, name string) (*models.Widget, error) {
    var w models.Widget
    err := r.pool.QueryRow(ctx,
        `INSERT INTO widgets (app_id, name) VALUES ($1, $2)
         RETURNING id, app_id, name, created_at, updated_at`,
        appID, name,
    ).Scan(&w.ID, &w.AppID, &w.Name, &w.CreatedAt, &w.UpdatedAt)
    return &w, err
}
```

Always use parameterized queries ($1, $2, etc.). Return nil for not-found cases.

## Step 4: Service

Create `internal/<domain>/service.go`. Business logic lives here, not in handlers:

```go
type Service interface {
    Create(appID uuid.UUID, req dto.CreateWidgetRequest) (*models.Widget, error)
}

type service struct {
    repo Repository
}

func NewService(repo Repository) Service {
    return &service{repo: repo}
}
```

If you need cross-domain dependencies (RBAC lookups, webhook dispatch, session management), use callback function fields rather than importing other domain packages:

```go
type service struct {
    repo            Repository
    LookupRoles     func(userID, appID uuid.UUID) ([]string, error)  // set in main.go
    WebhookService  webhook.Dispatcher                                // set in main.go
}
```

This is the established pattern for avoiding circular imports — see `internal/user/service.go` and `internal/social/service.go` for real examples.

## Step 5: Handler

Create `internal/<domain>/handler.go`. Every public handler method needs Swagger annotations:

```go
type Handler struct {
    Service Service
}

func NewHandler(service Service) *Handler {
    return &Handler{Service: service}
}

// @Summary Create a widget
// @Description Create a new widget for the current application
// @Tags widgets
// @Accept json
// @Produce json
// @Param X-App-ID header string true "Application ID"
// @Param request body dto.CreateWidgetRequest true "Widget data"
// @Success 201 {object} dto.WidgetResponse
// @Failure 400 {object} dto.ErrorResponse
// @Security ApiKeyAuth
// @Router /widgets [post]
func (h *Handler) Create(c *gin.Context) {
    // 1. Bind request
    // 2. Validate
    // 3. Call service
    // 4. Return response
}
```

Never expose raw database errors to clients — use generic messages or `pkg/errors/` types.

## Step 6: Wire in main.go

In `cmd/api/main.go`, follow the existing order:

1. Create repository: `widgetRepo := widget.NewRepository(pool)`
2. Create service: `widgetService := widget.NewService(widgetRepo)`
3. Wire cross-cutting concerns: `widgetService.WebhookService = webhookService`
4. Create handler: `widgetHandler := widget.NewHandler(widgetService)`
5. Register routes in the appropriate group:

```go
// Public routes (no auth)
public.POST("/widgets", widgetHandler.Create)

// Protected routes (JWT required)
protected.GET("/widgets", widgetHandler.List)

// Admin routes (admin auth required)
adminRoutes.DELETE("/widgets/:id", widgetHandler.Delete)
```

## Step 7: Regenerate Swagger

```bash
make swag-init
```

## Route Groups

- **Public**: No auth. Registration, login, OAuth callbacks, health checks.
- **Protected**: `middleware.AuthMiddleware()`. User-facing authenticated endpoints.
- **Admin**: Admin API key or admin session auth. Management endpoints.
- **Admin GUI**: HTMX-rendered pages with CSRF protection.

## Checklist

- [ ] Model has `AppID` for multi-tenancy (if applicable)
- [ ] DTOs have `validate`, `json`, and `example` tags
- [ ] Repository uses pgx pool + parameterized queries
- [ ] Service uses interface + constructor
- [ ] Cross-domain deps use callback fields, not direct imports
- [ ] Handler has full Swagger annotations
- [ ] Wired in `main.go` following existing order
- [ ] Route registered in correct group (public/protected/admin)
- [ ] `make swag-init` run after adding routes
- [ ] `make test` passes
- [ ] `make lint` passes
