FROM golang:1.24-bookworm AS builder

WORKDIR /app

# Install Node.js and pnpm
RUN curl -fsSL https://deb.nodesource.com/setup_20.x | bash - && \
    apt-get install -y nodejs && \
    npm install -g pnpm@10

COPY go.* ./
RUN go mod download

COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
RUN pnpm install

COPY . ./

# Use tools/build instead of direct go build
RUN chmod +x ./tools/build && ./tools/build

FROM debian:bookworm-slim
RUN set -x && apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y \
    ca-certificates git && \
    rm -rf /var/lib/apt/lists/*

COPY --from=builder /app/viz /app/viz

CMD ["/app/viz", "serve", "--port", "8080"]
