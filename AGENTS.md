# Sharelock

Sharelock is a small service for temporary secret sharing. It is
not a password manager or a full secret manager.

## Working agreements

- Start with a short plan (2–6 bullets), then execute. If unsure, ask at most 1–2 clarifying questions or propose 1–2
  options with tradeoffs.
- All comments and documentation should be in English.
- Add or adjust tests when behavior changes. If tests aren’t feasible, explain why and what was validated instead.
- When you introduce non-obvious behavior, leave a short comment or update the nearest README/AGENTS.md. Keep docs
  concise.
- After changes, check whether the project still builds and whether the relevant tests pass.
- Fix root causes, not symptoms. Do not stop at suppressing errors or adding defensive conditionals without addressing
  the underlying cause.
- Do not run all tests by default. For small changes, run only related tests.
- Do not explain obvious code changes unless asked.

## Product and security boundaries

- Encrypt and decrypt payloads in the browser. The server and SQLite database
  store ciphertext only; URL fragments (`#k=...`) must never be sent to the API.
- Keep the service self-contained: SQLite is the only required persistence
  dependency. Do not introduce Postgres, Redis, external queues, or CDNs.
- Treat changes to encryption format, share/revoke URL semantics, OpenAPI, or
  database migrations as compatibility-sensitive. Do not change them casually.
- Preserve the strict CSP: production frontend assets, fonts, and scripts must
  be served from `self`.

## Structure

- `cmd/sharelock` is the composition root: it loads configuration, opens SQLite,
  applies migrations, wires the HTTP API, and serves the embedded frontend.
- `internal/share` owns share operations, their HTTP routes, and SQLite-backed
  persistence behaviour.
- `internal/config`, `internal/errbrick`, and `internal/logbrick` provide
  process configuration and small cross-cutting building blocks; do not turn
  them into a home for feature code.
- `migration` owns the embedded SQLite schema migrations applied at startup.
- `cmd/openapi` generates the API document consumed by the frontend client.
- `frontend` is the Svelte + Vite client. Keep UI code feature-local; prefer
  small vertical slices over framework-heavy architectural layers. Its build
  output in `cmd/sharelock/assets/app` is embedded production output; never edit
  it by hand.

## Read on demand

- Before adding, changing, or debugging tests, read `docs/testing.md`.
- Before changing Go code, read `docs/code-style-and-conventions.md`.
- Before changing `frontend/**`, read `docs/frontend.md`.
- Before building frontend assets, changing embedded assets, or local runtime configuration, read `docs/development.md`.
