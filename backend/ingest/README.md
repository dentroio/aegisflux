# Ingest Service

A gRPC and HTTP service for ingesting events with streaming support, JSON Schema validation, and NATS publishing.

## Features

- gRPC server for streaming event ingestion
- HTTP health check, readiness, metrics, and visibility event endpoints
- Prometheus metrics collection
- JSON Schema validation for events
- NATS publishing with automatic reconnection
- JSON structured logging with slog
- Environment-based configuration
- Graceful shutdown handling

## Running Locally

### Prerequisites

- Go 1.21 or later
- Protocol Buffers compiler (`protoc`)
- NATS server (optional, for full functionality)

### Build and Run

```bash
# Navigate to the ingest service directory
cd backend/ingest

# Build the service
go build ./cmd/ingest

# Run the service
./ingest
```

### Environment Variables

- `INGEST_GRPC_ADDR`: gRPC server address (default: `:50051`)
- `INGEST_HTTP_ADDR`: HTTP server address (default: `:9090`)
- `NATS_URL`: NATS server URL (default: `nats://localhost:4222`)

### Example Usage

```bash
# Run with custom ports
INGEST_GRPC_ADDR=:9090 INGEST_HTTP_ADDR=:9091 ./ingest

# Run with custom NATS URL
NATS_URL=nats://nats-server:4222 ./ingest

# Check health endpoints
curl http://localhost:9090/healthz    # Health check (gRPC + NATS)
curl http://localhost:9090/readyz     # Readiness check (NATS + Schema)
curl http://localhost:9090/metrics    # Prometheus metrics
```

### Docker

The service includes a multi-stage Dockerfile that creates a minimal, secure container:

- **Base image**: `gcr.io/distroless/static-debian11` (minimal, no shell)
- **Build stage**: `golang:1.25-alpine` for compilation
- **Exposed ports**: 50051 (gRPC), 9090 (HTTP/metrics)
- **Size**: ~20MB final image

```bash
# Build Docker image
docker build -t aegisflux-ingest .

# Run with Docker
docker run -p 50051:50051 -p 9090:9090 \
  -e NATS_URL=nats://host.docker.internal:4222 \
  aegisflux-ingest

# Run with custom ports
docker run -p 50052:50051 -p 9091:9090 \
  -e NATS_URL=nats://host.docker.internal:4222 \
  aegisflux-ingest
```

### NATS Setup

To run with full NATS functionality:

```bash
# Install NATS server
brew install nats-server

# Start NATS server
nats-server

# Run the ingest service
./ingest
```

### gRPC Service

The service implements the `Ingest` service with a `PostEvents` streaming RPC:

```protobuf
service Ingest {
  rpc PostEvents(stream Event) returns (Ack);
}
```

### HTTP Visibility Event Ingest

Windows and macOS visibility agents can post Phase 1 visibility events as newline-delimited JSON or as a JSON array:

```bash
curl -sS -X POST \
  --data-binary @events.jsonl \
  http://localhost:9090/v1/visibility/events
```

Successful responses use HTTP `202`:

```json
{"ok":true,"accepted":169,"message":"visibility events accepted"}
```

The HTTP endpoint maps visibility envelopes to the existing ingest `Event` shape:

- `event_id` -> `id`
- `event_type` -> `type`
- `timestamp_ms` -> `timestamp`
- `source` -> `source`
- `payload` -> `payload`
- `device_id`, `agent_id`, `tenant_id`, `schema_version`, `sensor_version`, and `sequence` -> metadata

### Event Schema

Events are validated against a JSON Schema with the following requirements:

- **Required fields**: `id`, `type`, `source`, `timestamp`
- **Event types**: Any non-empty event type string, including `aegis.*` visibility event types
- **Timestamp**: Unix timestamp in milliseconds (minimum: 1)
- **Metadata**: Optional key-value string pairs
- **Payload**: Optional serialized payload string

### Health Endpoints

- **GET /healthz**: Returns `200 {"ok":true}` if both gRPC and NATS are healthy
- **GET /readyz**: Returns `200 {"ok":true}` when NATS is connected and schema is compiled
- **GET /metrics**: Returns Prometheus metrics in text format
- **POST /v1/visibility/events**: Accepts visibility JSONL or JSON arrays, publishes accepted events to NATS, and appends them to the local visibility event store
- **GET /v1/visibility/events**: Returns recent stored visibility events, optionally filtered by `event_id`, `device_id`, `agent_id`, `event_type`, and `limit`
- **GET /v1/visibility/devices**: Returns device summaries derived from stored visibility events, optionally filtered by `tenant_id` and `limit`
- **GET /v1/visibility/processes**: Returns normalized process start/end events, optionally filtered by `device_id`, `agent_id`, `process_guid`, `pid`, and `limit`
- **GET /v1/visibility/flows**: Returns normalized flow start/end events, optionally filtered by `device_id`, `agent_id`, `flow_id`, `process_guid`, `pid`, `remote_ip`, `remote_hostname`, and `limit`
- **GET /v1/visibility/dns**: Returns normalized DNS observations, optionally filtered by `device_id`, `agent_id`, `process_guid`, `pid`, `query`, `answer`, and `limit`
- **GET /v1/visibility/findings**: Returns normalized AI-agent detections and risk findings, optionally filtered by `device_id`, `agent_id`, `process_guid`, `flow_id`, `detection_id`, `finding_id`, `severity`, and `limit`
- **GET /v1/visibility/investigation**: Returns a combined process, flow, DNS, and findings investigation path for a `device_id`, optionally narrowed by `agent_id`, `process_guid`, `pid`, and `limit`
- **GET /v1/clarion/events**: Exports stored Aegis visibility events in the `aegis-clarion.export.v1` lab contract, optionally filtered by `tenant_id`, `device_id`, `agent_id`, `event_id`, `event_type`, and `limit`

### Prometheus Metrics

- **events_total**: Total number of events processed successfully
- **events_invalid_total**: Total number of invalid events rejected
- **nats_publish_errors_total**: Total number of NATS publish errors

### Local Visibility Event Store

The ingest service keeps a durable lab event log for Phase 1 visibility validation.

- **JSONL (default):** set `AEGIS_VISIBILITY_STORE_PATH` to control the file location. If unset, the service writes to `data/visibility-events.jsonl` relative to its working directory.
- **SQLite (durable, query-friendly):** set `AEGIS_VISIBILITY_SQLITE_PATH` to a `.db` file path. When this variable is non-empty, ingest uses SQLite with WAL instead of JSONL (JSONL path is ignored).

Query recent stored events:

```bash
curl -sS 'http://localhost:9090/v1/visibility/events?device_id=RMARTINEZ-WS&limit=50'
curl -sS 'http://localhost:9090/v1/visibility/processes?device_id=RMARTINEZ-WS&limit=50'
curl -sS 'http://localhost:9090/v1/visibility/flows?device_id=RMARTINEZ-WS&pid=9324&limit=50'
curl -sS 'http://localhost:9090/v1/visibility/dns?device_id=RMARTINEZ-WS&query=api.model-gateway.lab&limit=50'
curl -sS 'http://localhost:9090/v1/visibility/findings?device_id=RMARTINEZ-WS&process_guid=windows-dev-agent-01:1777062000000:9324&limit=50'
curl -sS 'http://localhost:9090/v1/visibility/investigation?device_id=RMARTINEZ-WS&process_guid=windows-dev-agent-01:1777062000000:9324&limit=50'
```

The JSONL store loads all events into memory for queries. The SQLite store keeps events on disk and is better for sustained lab runs. Production may still move to a fleet-scale database model later.

### Clarion Lab Export

Aegis and Clarion remain independent products. For Phase 1 lab integration, Clarion can pull a contract-shaped export from the Aegis ingest visibility store without reading Aegis internals:

```bash
curl -sS 'http://localhost:9090/v1/clarion/events?tenant_id=tenant-a&limit=100'
```

The response preserves the raw Aegis visibility event envelope and payload, adds `contract_version: aegis-clarion.export.v1`, and includes `clarion_context_objects` hints such as `Device`, `Agent`, `Process`, `Flow`, `Destination`, or `AI-agent or automation finding`. This endpoint is intended for observe-only contract validation before a production event transport is selected.

### NATS Publishing

Events are published to NATS with:

- **Subject**: `events.raw`
- **Headers**: 
  - `x-host-id`: Host ID from event metadata (if present)
  - `x-event-id`: Event ID
  - `x-event-type`: Event type
  - `x-event-source`: Event source
  - `x-timestamp`: Event timestamp
- **Payload**: JSON-encoded Event

### Development

To regenerate protobuf stubs:

```bash
protoc --go_out=. --go_opt=paths=source_relative \
       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
       protos/ingest.proto
```

To run tests:

```bash
# Run all tests
go test ./...

# Run specific package tests
go test ./internal/validate/ -v
go test ./internal/nats/ -v
```

Manual smoke utilities live under `cmd/` so they compile cleanly with the rest of the module:

```bash
go run ./cmd/test-logging
go run ./cmd/test-metrics
go run ./cmd/test-nats
```

Run the repeatable visibility investigation smoke test with a fixture-backed process -> flow -> DNS -> findings path:

```bash
./scripts/smoke_visibility_investigation.sh
```

The script requires a reachable NATS server at `NATS_URL` and uses test ports `127.0.0.1:19090` and `127.0.0.1:15051` by default.

## Architecture

- `cmd/ingest/main.go`: Application entry point
- `internal/server/grpc.go`: gRPC server implementation
- `internal/server/http.go`: HTTP visibility event ingest and query implementation
- `internal/server/visibility_store.go`: JSONL-backed lab visibility event store
- `internal/validate/schema.go`: JSON Schema validation
- `internal/nats/publish.go`: NATS publishing
- `internal/health/`: Health check endpoints and status management
- `internal/metrics/`: Prometheus metrics collection
- `protos/`: Protocol buffer definitions and generated code
- `schemas/`: JSON Schema definitions

## Error Handling

- **Validation errors**: Returns `InvalidArgument` gRPC status with warning logs
- **Publishing errors**: Returns `Unavailable` gRPC status with error logs
- **NATS connection**: Fails fast on startup if NATS unavailable
- **Timeouts**: 2-second timeout for per-event publishing
- **Graceful shutdown**: Handles SIGINT/SIGTERM signals and closes NATS connection

## Logging

- **Structured logging**: All logs include `event_id`, `event_type`, and `host_id` fields
- **Event processing**: Info-level logs for each event processed
- **Validation failures**: Warning-level logs with detailed error information
- **Publishing failures**: Error-level logs with context and error details
- **Graceful shutdown**: Info-level logs for connection cleanup
