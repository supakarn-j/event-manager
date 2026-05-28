# Event Manager

Event Manager is a small web application for inspecting and managing Redis Streams. It provides a Go API server, a React/Vite frontend, and live updates over WebSocket when stream events or consumer health data change.

## Features

- List Redis streams and basic stream metadata.
- Inspect stream events, payload fields, and acknowledgement status.
- Create streams and publish JSON events into a stream.
- Delete streams, events, and stream consumers.
- Watch stream changes and consumer health updates in real time through `/ws`.

## Tech Stack

- Go 1.25.6
- Gin HTTP framework
- Redis Streams
- Centrifuge WebSocket server
- React 19, TypeScript, Vite, and Tailwind CSS

## Requirements

- Go 1.25 or newer
- Node.js and npm
- Redis running on `localhost:6379`
- Optional: `air` for live backend reloads when using `make run`
- Optional: Docker or Podman for container builds

The application currently connects to Redis with this hardcoded address:

```text
localhost:6379
```

If you run the app inside a container, Redis must be reachable from inside that container as `localhost:6379`, or the code must be updated to read Redis connection settings from environment variables.

## Redis Setup

Start Redis locally:

```bash
redis-server
```

Enable Redis keyspace/keyevent notifications so Event Manager can publish live stream and consumer health updates:

```bash
redis-cli CONFIG SET notify-keyspace-events Ehtx
```

Useful flags:

```text
E  Keyevent notifications, published with the __keyevent@<db>__ prefix.
h  Hash command events, such as HSET.
t  Stream command events, such as XADD.
x  Expired events.
```

## Configuration

The server reads environment variables from the shell and from a local `.env` file when present.

| Variable | Default | Description |
| --- | --- | --- |
| `ENV` | `local` | Runtime environment label. Supported values in code are `local`, `dev`, and `prd`. |
| `PORT` | `8080` | HTTP port for the Go server. |
| `LOG_LEVEL` | `INFO` | Logger level. Supported values are `DEBUG`, `INFO`, `WARN`, `ERROR`, `FATAL`, and `PANIC`. |

Example `.env`:

```env
ENV=local
PORT=8080
LOG_LEVEL=DEBUG
```

## Run Locally

Install frontend dependencies:

```bash
cd frontend
npm install
```

Build the frontend bundle. The Go server embeds `frontend/dist`, so this step must run before building or running the backend binary:

```bash
npm run build
cd ..
```

Download Go dependencies and run the server:

```bash
go mod download
go run .
```

Open the application:

```text
http://localhost:8080
```

Health check:

```bash
curl http://localhost:8080/api/v1/healthz
```

## Development

Run frontend checks and builds from the `frontend` directory:

```bash
npm run lint
npm run build
```

Run Go tests:

```bash
go test ./...
```

Build the frontend with the Makefile helper:

```bash
make frontend-build
```

Build the Go binary:

```bash
make build
```

The binary is written to:

```text
bin/eventmanager
```

Run with `air` if it is installed:

```bash
make run
```

## API Overview

Base path:

```text
/api/v1
```

Common endpoints:

| Method | Path | Description |
| --- | --- | --- |
| `GET` | `/api/v1/healthz` | Health check. |
| `GET` | `/api/v1/streams` | List Redis streams. |
| `POST` | `/api/v1/streams` | Create a stream. Send form field `name`. |
| `GET` | `/api/v1/streams/:stream` | Get stream consumer group details. |
| `GET` | `/api/v1/streams/:stream/events` | List stream events. |
| `POST` | `/api/v1/streams/:stream/events` | Publish an event. Body must be a JSON object. |
| `DELETE` | `/api/v1/streams/:stream/events/:id` | Delete an event and its ack metadata. |
| `DELETE` | `/api/v1/streams/:stream/consumers/:group/:name` | Delete a consumer from a group. |
| `DELETE` | `/api/v1/streams/:stream` | Delete a stream and its ack metadata. |
| `GET` | `/ws` | WebSocket endpoint for live updates. |

Example event publish:

```bash
curl -X POST http://localhost:8080/api/v1/streams/orders/events \
  -H 'Content-Type: application/json' \
  -d '{"order_id":"A1001","status":"created"}'
```

## Build For Production

Build the frontend first:

```bash
cd frontend
npm ci
npm run build
cd ..
```

Build the backend binary:

```bash
go mod download
GOOS=linux GOARCH=amd64 go build -o bin/eventmanager .
```

Run the binary:

```bash
PORT=8080 ENV=prd ./bin/eventmanager
```

## Build A Container Image

This repository does not currently include a Dockerfile. Add the following `Dockerfile` at the repository root if you want to build a container image:

```Dockerfile
FROM node:25-alpine AS frontend
WORKDIR /src/frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

FROM golang:1.25-alpine AS backend
WORKDIR /src
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /src/frontend/dist ./frontend/dist
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/eventmanager .

FROM alpine:3.22
WORKDIR /app
RUN adduser -D -H appuser
COPY --from=backend /out/eventmanager /app/eventmanager
USER appuser
EXPOSE 8080
ENV ENV=prd
ENV PORT=8080
ENTRYPOINT ["/app/eventmanager"]
```

Build the image:

```bash
docker build -t event-manager:local .
```

Run the image:

```bash
docker run --rm -p 8080:8080 --network host event-manager:local
```

`--network host` is used here because the application connects to Redis at `localhost:6379`. On platforms where host networking is unavailable or undesirable, update the application to use a configurable Redis address before running it in a separate container network.

## Troubleshooting

- `pattern frontend/dist/*: no matching files found`: run `cd frontend && npm run build` before building the Go server.
- No live updates appear: confirm Redis notifications are enabled with `redis-cli CONFIG GET notify-keyspace-events`.
- Cannot connect to Redis: make sure Redis is running on `localhost:6379` from the same environment where the app is running.
- `make run` fails because `air` is missing: install `air` or run `go run .` directly.
