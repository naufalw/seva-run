FROM golang:1.25-bookworm AS builder
WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download


COPY . ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/judge

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    g++ bash ca-certificates time && \
    rm -rf /var/lib/apt/lists/*

RUN useradd -m runner
USER runner
WORKDIR /home/runner

COPY --from=builder /out/judge /usr/local/bin/judge

ENV GIN_MODE=release
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/judge"]