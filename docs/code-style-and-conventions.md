## Coding Style & Naming Conventions

- Keep functions flat. Prefer guard clauses and early returns over nested control flow.

## Go Coding Style & Naming Conventions

- Use singular names for packages.
- Package names are short, lowercase, and business-oriented.
- Go packages should be self-contained and have minimal external dependencies.
- Export only identifiers that are needed outside the package.
- Go 1.27 module: prefer modern Go features that fit naturally.
- Indentation follows Go defaults (tabs in Go files).
- Keep CLI/config/logging naming consistent with existing patterns.
- All lint and format rules are defined in `.golangci.yml`.
- Keep semantic validation in domain objects/use cases. HTTP schemas should only validate transport shape, required
  fields, nullability, and wire types.

### Error conventions

- Use shared sentinels from `internal/errbrick` for cross-layer error classification.
- Wrap low-level errors with context using `fmt.Errorf(...: %w, err)`.
