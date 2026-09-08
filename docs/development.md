# Development

## Local development and verification

```sh
DB_PATH=./data/sharelock.db go run ./cmd/sharelock
npm --prefix frontend run dev

npm --prefix frontend run check
npm --prefix frontend run build
go test ./...
```

Before changing code, inspect the affected route, use case, generated-client
contract, and callers. Keep changes narrowly scoped and run the most relevant
checks after editing.
