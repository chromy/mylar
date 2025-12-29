FROM golang:1.24-bookworm AS builder

WORKDIR /app

COPY go.* ./
RUN go mod download

COPY . ./

RUN curl -L -o cpdf https://github.com/coherentgraphics/cpdf-binaries/raw/master/Linux-Intel-64bit/cpdf

RUN go build -v -o byte cmd/main.go

FROM debian:bookworm-slim
RUN set -x && apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y \
    ca-certificates ghostscript libvips-tools poppler-utils && \
    rm -rf /var/lib/apt/lists/*

COPY --from=builder /app/byte /app/byte
COPY --from=builder /app/cpdf /usr/local/bin/cpdf
ENV PATH="/usr/local/bin:${PATH}"
RUN chmod +x /usr/local/bin/cpdf

CMD ["/app/byte", "serve"]
