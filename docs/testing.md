# Testing

```bash
make test

go test -v ./internal/user -run TestRegister
go test -v ./internal/user -run TestRegister -count=1
go test -v -race ./internal/user
go test -cover ./...
go test -v -tags=integration ./app/...
```

Tests sit next to the code they cover (`service_test.go` beside `service.go`). Public API lifecycle tests are in `app/app_integration_test.go` and need the `integration` build tag.

For a running reference server, use Swagger UI at `/swagger/index.html`.

## Before a commit

```bash
make ci
```

That is fmt, lint, test, gosec, govulncheck, and a production build. Run `make swag-init` as well if you changed HTTP handlers.
