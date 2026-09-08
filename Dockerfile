ARG GO_VERSION=1.27
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_TIME=unknown

FROM golang:${GO_VERSION}-alpine AS builder
ARG VERSION
ARG COMMIT
ARG BUILD_TIME
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath \
  -ldflags="-s -w -X 'github.com/demeero/sharelock/internal/config.buildVersion=${VERSION}' -X 'github.com/demeero/sharelock/internal/config.commit=${COMMIT}' -X 'github.com/demeero/sharelock/internal/config.buildTime=${BUILD_TIME}'" \
  -o /out/sharelock ./cmd/sharelock

FROM alpine:3.22 AS runner

RUN addgroup -S app && adduser -S app -G app \
  # The default DB_PATH resolves to /data/sharelock.db. Ship the directory
  # owned by app so Docker copies that ownership into an empty named volume
  # mounted there; otherwise it is created as root and the app cannot write.
  && mkdir -p /data && chown app:app /data

COPY --from=builder /out/sharelock /usr/local/bin/sharelock

USER app
ENTRYPOINT ["/usr/local/bin/sharelock"]
