FROM golang:1.23 AS build
WORKDIR /app
COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal
RUN go build -o /quicknotes ./cmd/server

FROM debian:bookworm-slim
WORKDIR /app
RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates wget \
    && rm -rf /var/lib/apt/lists/*
COPY --from=build /quicknotes /usr/local/bin/quicknotes
COPY static ./static
COPY migrations ./migrations
ENV QUICKNOTES_DATA_FILE=/app/data/notes.json
EXPOSE 8080
HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=5 CMD wget -qO- http://127.0.0.1:8080/healthz || exit 1
CMD ["quicknotes"]
