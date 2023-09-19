FROM ubuntu AS builder

RUN apt update && \
    apt install -y openssl git vim curl make build-essential && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY . .
SHELL ["/bin/bash", "-c"]
RUN GIT_COMMIT=$(git rev-parse --short HEAD) && \
  CGO_ENABLED=1 GOOS=linux ./bin/go build \
    -ldflags "-X main.GitCommit=$GIT_COMMIT" \
    -o /app/wkr-gh-sponsor ./cmd/wkr-gh-sponsor

FROM ubuntu as runtime
RUN apt update && \
    apt install -y ca-certificates && \
    rm -rf /var/lib/apt/lists/*
COPY --from=builder /app/wkr-gh-sponsor /app/wkr-gh-sponsor
WORKDIR /app