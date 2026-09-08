# Sharelock

Sharelock is a small, self-hosted service for sharing encrypted secrets, configuration files, certificates, and small files. It is deliberately **not** a password manager or a general secret manager.

The binary applies its embedded database migrations on every startup before it begins serving requests.

The browser encrypts every item before upload. The SQLite database stores only an opaque encrypted bundle; the decryption key lives after `#` in the share URL and is never sent in an HTTP request.

## What includes

- a single Go binary and an embedded SQLite database;
- browser-side AES-256-GCM encryption using the Web Crypto API;
- multiple text or file items in one share;
- fixed expiration and optional burn-after-first-open;
- a separate revoke capability returned only at creation time;
- a small terminal-style Web UI, compiled to static assets and embedded in the Go binary.

It intentionally does not include accounts, teams, RBAC, search, revisions, audit trails, password vaulting, secret rotation, or external infrastructure.

## Installation Options

You can run Sharelock in three ways:

- use the published Docker Hub image: `docker.io/demeero/sharelock`
- download a prebuilt binary from [GitHub Releases](https://github.com/demeero/sharelock/releases)
- build it locally from this repository

## Running the Published Docker Hub Image

Images are published to `docker.io/demeero/sharelock` on every tagged release (see [Releases and Conventional Commits](#releases-and-conventional-commits)). Available tags:

- `latest` — most recent release
- `vX.Y.Z`, `vX.Y`, `vX` — pinned to a specific release, minor line, or major line

```sh
docker pull demeero/sharelock:latest
```

The container has no `WORKDIR`, so the default `DB_PATH` (`./data/sharelock.db`) resolves to `/data/sharelock.db`. Mount a volume there to persist the database across restarts:

```sh
docker run -d \
  --name sharelock \
  -p 8080:8080 \
  -v sharelock-data:/data \
  demeero/sharelock:latest
```

The process runs as a non-root user (`uid=100`, `gid=101`), so a bind-mounted host directory must be writable by that user, e.g.:

```sh
mkdir -p ./data && chown 100:101 ./data
docker run -d \
  --name sharelock \
  -p 8080:8080 \
  -v "$(pwd)/data:/data" \
  demeero/sharelock:latest
```

Pass configuration as environment variables (see [Configuration](#configuration)):

```sh
docker run -d \
  --name sharelock \
  -p 8080:8080 \
  -v sharelock-data:/data \
  -e LOG_LEVEL=debug \
  -e SHARE_MAX_TTL=168h \
  demeero/sharelock:latest
```

For many variables, an env file is easier to manage than repeated `-e` flags. Create one (same `KEY=value` format as the optional local `.env` file described in [Environment files](#environment-files)):

```dotenv
# sharelock.env
LOG_LEVEL=debug
SHARE_MAX_TTL=168h
SHARE_MAX_ENCRYPTED_BYTES=5242880
```

Pass it to `docker run` with `--env-file`; Docker loads it into the container's environment before the process starts:

```sh
docker run -d \
  --name sharelock \
  -p 8080:8080 \
  -v sharelock-data:/data \
  --env-file ./sharelock.env \
  demeero/sharelock:latest
```

`--env-file` is a Docker-level mechanism and is independent of the application's own `.env`/`ENV_FILE` support; don't mix the two inside a container, and don't commit an env file containing deployment-specific values.

To terminate TLS in the container itself, mount the certificate and key and point the corresponding variables at them:

```sh
docker run -d \
  --name sharelock \
  -p 8443:8443 \
  -v sharelock-data:/data \
  -v "$(pwd)/certs:/certs:ro" \
  -e HTTP_ADDR=:8443 \
  -e TLS_CERT_FILE=/certs/server.crt \
  -e TLS_CERT_KEY_FILE=/certs/server.key \
  demeero/sharelock:latest
```

## Running a Prebuilt Binary

Every tagged release publishes archives for Linux, macOS, and Windows (amd64/arm64) to [GitHub Releases](https://github.com/demeero/sharelock/releases). Each archive contains the `sharelock` binary (`sharelock.exe` on Windows) alongside `LICENSE` and `README.md`; the release also includes a `checksums.txt` with the SHA-256 sum of every archive.

```sh
curl -LO https://github.com/demeero/sharelock/releases/download/vX.Y.Z/sharelock_vX.Y.Z_linux_amd64.tar.gz
curl -LO https://github.com/demeero/sharelock/releases/download/vX.Y.Z/checksums.txt
sha256sum --ignore-missing -c checksums.txt
tar -xzf sharelock_vX.Y.Z_linux_amd64.tar.gz
./sharelock
```

The binary applies its embedded database migrations and serves the embedded Web UI on startup, same as the Docker image. Configure it via [environment variables](#configuration) or a local [`.env` file](#environment-files).

## Configuration

Sharelock reads its runtime configuration from environment variables.

| Variable                    | Description                                                                                          | Default               |
| --------------------------- | ---------------------------------------------------------------------------------------------------- | --------------------- |
| `DB_PATH`                   | Path to the SQLite database file. Its parent directory is created on startup.                        | `./data/sharelock.db` |
| `DB_STARTUP_TIMEOUT`        | Maximum time to wait for the database connection during startup.                                     | `10s`                 |
| `DB_MAX_OPEN_CONNS`         | Maximum number of open SQLite connections.                                                           | `4`                   |
| `DB_MAX_IDLE_CONNS`         | Maximum number of idle SQLite connections.                                                           | `4`                   |
| `HTTP_ADDR`                 | Address on which the HTTP server listens.                                                            | `:8080`               |
| `HTTP_READ_HEADER_TIMEOUT`  | Maximum time to read request headers.                                                                | `10s`                 |
| `HTTP_READ_TIMEOUT`         | Maximum time to read a complete request.                                                             | `30s`                 |
| `HTTP_WRITE_TIMEOUT`        | Maximum time to write an HTTP response.                                                              | `30s`                 |
| `HTTP_IDLE_TIMEOUT`         | Maximum time to keep an idle HTTP connection open; also used as the graceful HTTP shutdown deadline. | `60s`                 |
| `HTTP_SHUTDOWN_TIMEOUT`     | HTTP server shutdown timeout                                                                         | `10s`                 |
| `TLS_CERT_FILE`             | Path to the TLS certificate PEM file. Must be set together with `TLS_CERT_KEY_FILE`.                 | Disabled              |
| `TLS_CERT_KEY_FILE`         | Path to the TLS private-key PEM file. Must be set together with `TLS_CERT_FILE`.                     | Disabled              |
| `SHARE_MAX_ENCRYPTED_BYTES` | Maximum ciphertext bundle size accepted when creating a share.                                       | `10485760` (10 MiB)   |
| `SHARE_MAX_TTL`             | Maximum lifetime that may be requested for a share.                                                  | `720h` (30 days)      |
| `SHARE_VACUUM_INTERVAL`     | Interval for removing expired or consumed shares.                                                    | `1h`                  |
| `SHARE_IDENTIFIER_SIZE`     | Number of random bytes in each share and revoke identifier before URL-safe Base64 encoding.          | `32`                  |
| `LOG_LEVEL`                 | Minimum structured log level. Invalid values fall back to `info`.                                    | `info`                |
| `LOG_ADD_SOURCE`            | Include source file and line information in structured logs.                                         | `false`               |
| `SHUTDOWN_TIMEOUT`          | Service shutdown timeout.                                                                            | `15s`                 |

Durations use the [Go duration format](https://pkg.go.dev/time#ParseDuration), such as `5s`, `1m`, or `24h`. All connection limits, share limits, identifier sizes, and durations must be positive. Built-in TLS is enabled only when both TLS file variables are non-empty.

### Environment files

The binary optionally reads a `.env` file from its current working directory. Use one `KEY=value` entry per line; blank lines and lines beginning with `#` are ignored.

```dotenv
DB_PATH=./data/sharelock.db
HTTP_ADDR=127.0.0.1:8080
LOG_LEVEL=debug
```

Start the service from the directory that contains the file:

```sh
go run ./cmd/sharelock
```

For a non-default location, set `ENV_FILE` to the exact file path. Unlike the optional local `.env`, this file must exist and be readable:

```sh
ENV_FILE=/etc/sharelock/sharelock.env ./sharelock
```

Non-empty environment variables already supplied to the process take precedence over entries in either file, so deployments can override individual values without modifying the file. Do not commit `.env` files containing deployment-specific values.

## Security boundary

The service protects a stolen SQLite file from revealing content, because it contains ciphertext only. It does **not** protect against a malicious or compromised live server serving altered JavaScript; Use HTTPS, keep all static assets local, and share the URL only through a trusted channel.

“Burn after first open” guarantees that Sharelock will not serve the blob again after the first successful request. It cannot physically erase bytes already present in a filesystem snapshot, SQLite WAL, or backup; apply a short backup retention policy when that matters.

## Releases and Conventional Commits

Releases are managed locally via `task release`, which runs `./release.sh`. The script is responsible for:

- Generating/Updating `CHANGELOG.md` (via git-cliff)
- Creating an annotated tag (SemVer, typically `vX.Y.Z`)

After that, pushing the release commit and tag triggers GitHub Actions, which publishes the Docker image and creates a [GitHub Release](https://github.com/demeero/sharelock/releases) with prebuilt binaries for Linux, macOS, and Windows.

Conventional Commits are strongly recommended because they keep history readable and improve automated changelog generation.

### Format

```
<type>[(scope)]: <subject>
```

### Types

- `feat`: new functionality
- `fix`: bug fix
- `refactor`: code change without behavior change
- `style`: changes that do not affect the meaning of the code (white-space, formatting, missing semi-colons, etc)
- `perf`: code change that improves performance
- `docs`: documentation only
- `chore`: tooling/maintenance
- `test`: adding/updating tests
- `ci`: CI/CD changes

### Scope

Use a short, lowercase scope describing the area, e.g. `cli`, `api`, `deploy`, `config`.
Scope can also include a ticket number or a combination of both, for example: `feat(cli-123): commit message`.

### References

- https://www.conventionalcommits.org/en/v1.0.0/
- https://git-cliff.org/docs/
