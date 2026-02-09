# Transjakarta Code Test

Fleet vehicle location tracking system built with Go. Receives real-time vehicle locations via MQTT, stores them in PostgreSQL, exposes REST APIs, and triggers geofence alerts via RabbitMQ.

## Tech Stack

| Technology | Role |
|---|---|
| **Go (Gin)** | HTTP server and business logic |
| **MQTT (Mosquitto)** | Vehicle location ingestion protocol |
| **PostgreSQL** | Persistent storage for location data |
| **RabbitMQ** | Internal event bus for geofence alerts |
| **Docker Compose** | Infrastructure orchestration |

## Architecture

### High-Level Design

```
[Vehicles / Publisher]
        |
        | MQTT publish /fleet/vehicle/{id}/location
        v
  +-------------+
  | MQTT Broker  |  (Mosquitto :1883)
  +------+------+
         | subscribe
         v
  +-------------+
  |   Server     |
  | (Go / Gin)  |
  +--+-------+--+
     |       |
     |       | geofence triggered (within 50m radius)
     v       v
  +------+ +----------+
  |Postgres| | RabbitMQ  |
  |(:5432) | |(:5672)    |
  +------+ +-----+----+
                  | consume
                  v
           +--------------+
           |Event Listener |
           +--------------+
```

**Flow:**
1. Vehicles (or the mock publisher) send location data as JSON to the MQTT broker on topic `/fleet/vehicle/{vehicle_id}/location`
2. The server subscribes to MQTT, validates the payload, and persists the location to PostgreSQL
3. On each location update, the server checks if the vehicle is within 50m of any configured geofence point (using the Haversine formula)
4. If a geofence is triggered, the server publishes an alert to RabbitMQ (`fleet.events` exchange -> `geofence_alerts` queue)
5. The event listener consumes alerts from RabbitMQ and logs them

**Why MQTT + RabbitMQ (two message systems)?**
- **MQTT** is the standard protocol for IoT/vehicle devices — lightweight, supports QoS, designed for unreliable networks. It handles the **external boundary** (vehicles -> server)
- **RabbitMQ** is used for **internal event processing** between services — reliable queue semantics, exchange routing, consumer acknowledgment. Geofence alerts are internal domain events, not IoT telemetry

### Project Structure

```
tj-test/
├── cmd/
│   ├── server/              # Main server entrypoint (DI wiring, startup)
│   ├── publisher/           # Mock MQTT publisher for testing
│   └── event_listener/      # RabbitMQ geofence alert consumer
├── config/                  # Shared infrastructure clients
│   ├── env.go               # Environment variable loading
│   ├── postgres.go          # PostgreSQL connection
│   ├── rabbitmq.go          # RabbitMQ connection
│   ├── mqtt.go              # MQTT client
│   └── health.go            # Health check endpoint
├── module/
│   └── core/
│       ├── build.go         # Module DI wiring
│       ├── domain/          # Domain types (the contract)
│       │   ├── vehicle.go
│       │   ├── location.go
│       │   └── geofence.go
│       ├── service/         # Business logic (public)
│       │   ├── location.go
│       │   └── geofence.go
│       └── internal/        # Implementation details (Go-enforced private)
│           ├── handler/
│           │   ├── http/        # Gin HTTP handlers
│           │   └── subscriber/  # MQTT subscriber
│           └── repository/
│               ├── database/    # DB interface + Postgres impl
│               └── publisher/   # Publisher interface + RabbitMQ impl
├── migrations/              # SQL migration files
├── scripts/                 # Integration test script
├── docker-compose.yml
├── Dockerfile
└── Makefile
```

**Why this structure?**

| Decision | Reason |
|---|---|
| **`config/` at root** | Shared infrastructure clients, reusable by any module |
| **`module/core/`** | Modular design — each module owns its domain, service, and internal implementation. Adding a new module (e.g. `module/analytics/`) is isolated |
| **`domain/` is public** | Domain types are the contract between all layers and modules. Every interaction uses domain types |
| **`service/` is public** | Business logic is accessible to other modules — they just need to satisfy the dependency contract |
| **`internal/` is private** | Handlers and repositories are implementation details. Go compiler enforces that nothing outside `core/` can import `core/internal/...` |
| **`build.go` per module** | Single DI entry point. `main.go` only calls `core.Build(...)` — never reaches into module internals |
| **Interfaces at consumer** | Each handler defines the interface it needs (interface segregation). Services and repos are mockable for unit testing |

### Clean Architecture Principles

- **Domain types are the transfer contract** — services return `domain.VehicleLocation`, not DTOs. Handlers at the boundary transform domain types into transport-specific formats (JSON responses, MQTT payloads)
- **No leaky abstractions** — MQTT payload struct lives in the subscriber handler, HTTP response struct lives in the HTTP handler. Domain knows nothing about transport format
- **Validation at boundaries** — the MQTT subscriber validates incoming JSON before converting to domain types. The service layer trusts it receives valid domain objects
- **Dependency inversion** — handlers depend on service interfaces, services depend on repository interfaces. All wiring happens in `build.go`

## API Contract

### Health Check

```
GET /healthz
```

Response `200 OK` (all healthy) or `503 Service Unavailable`:

```json
{
  "status": "healthy",
  "dependencies": {
    "postgres": { "status": "up" },
    "rabbitmq": { "status": "up" },
    "mqtt": { "status": "up" }
  }
}
```

### Get All Vehicles

```
GET /vehicles
```

Response `200 OK`:

```json
[
  { "vehicle_id": "B1234XYZ" },
  { "vehicle_id": "B5678ABC" }
]
```

### Get Latest Vehicle Location

```
GET /vehicles/{vehicle_id}/location
```

Response `200 OK`:

```json
{
  "vehicle_id": "B1234XYZ",
  "latitude": -6.2088,
  "longitude": 106.8456,
  "timestamp": 1715003456
}
```

Response `404 Not Found`:

```json
{
  "error": "vehicle not found"
}
```

### Get Vehicle Location History

```
GET /vehicles/{vehicle_id}/history?start={unix_ts}&end={unix_ts}
```

Response `200 OK`:

```json
[
  {
    "vehicle_id": "B1234XYZ",
    "latitude": -6.2088,
    "longitude": 106.8456,
    "timestamp": 1715003456
  },
  {
    "vehicle_id": "B1234XYZ",
    "latitude": -6.2090,
    "longitude": 106.8460,
    "timestamp": 1715003458
  }
]
```

### MQTT Payload (Inbound)

Topic: `/fleet/vehicle/{vehicle_id}/location`

```json
{
  "vehicle_id": "B1234XYZ",
  "latitude": -6.2088,
  "longitude": 106.8456,
  "timestamp": 1715003456
}
```

Validation rules:
- `vehicle_id` — required, non-empty
- `latitude` — required, between -90 and 90
- `longitude` — required, between -180 and 180
- `timestamp` — required, positive integer (unix epoch)

### RabbitMQ Geofence Alert (Outbound)

Exchange: `fleet.events` (fanout) | Queue: `geofence_alerts`

```json
{
  "vehicle_id": "B1234XYZ",
  "event": "geofence_entry",
  "location": {
    "latitude": -6.2088,
    "longitude": 106.8456
  },
  "timestamp": 1715003456
}
```

## Database Schema

```sql
CREATE TABLE vehicle_locations (
    id BIGSERIAL PRIMARY KEY,
    vehicle_id VARCHAR(50) NOT NULL,
    latitude DOUBLE PRECISION NOT NULL,
    longitude DOUBLE PRECISION NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_vehicle_locations_vehicle_id_timestamp
    ON vehicle_locations (vehicle_id, timestamp DESC);
```

## Getting Started

### Prerequisites

- Go 1.21+
- Docker & Docker Compose

### 1. Start Infrastructure

```bash
make infra
```

This starts PostgreSQL, Mosquitto (MQTT), and RabbitMQ with health checks. The migration runs automatically on Postgres startup.

### 2. Run the Server

```bash
make run
```

The server starts on `:8080`, connects to all dependencies, subscribes to MQTT, and serves the REST API.

### 3. Seed Mock Data

In a separate terminal:

```bash
make publisher INTERVAL=2
```

Publishes random vehicle locations every 2 seconds. Generates 5 random vehicle IDs at startup and reuses them. ~30% of messages land near the geofence point to trigger alerts.

### 4. Listen to Geofence Events

In a separate terminal:

```bash
make event-listener
```

Consumes and logs geofence alerts from RabbitMQ.

### 5. Query the API

```bash
# All vehicles
curl http://localhost:8080/vehicles

# Latest location
curl http://localhost:8080/vehicles/{vehicle_id}/location

# Location history
curl "http://localhost:8080/vehicles/{vehicle_id}/history?start=1715000000&end=1715009999"

# Health check
curl http://localhost:8080/healthz
```

## Testing

### Unit Tests

```bash
make test
```

Runs 34 unit tests across all layers with mocked dependencies:
- HTTP handler tests (Gin test mode + mock service)
- MQTT subscriber tests (fake MQTT message + mock services)
- Service tests (mock repository interfaces)
- Postgres repository tests (go-sqlmock)
- Geofence service tests (haversine calculation + mock publisher)

### Integration Tests

```bash
make integration-test
```

End-to-end test script that:
1. Verifies Docker Compose services are running
2. Seeds 5 random vehicles via MQTT
3. Checks PostgreSQL for persisted data
4. Tests all API endpoints with seeded data
5. Triggers and verifies geofence alerts in RabbitMQ

### Linting

```bash
make lint   # golangci-lint
make fmt    # gofmt
```

## Environment Variables

| Variable | Default | Description |
|---|---|---|
| `POSTGRES_DSN` | `postgres://postgres:postgres@localhost:5432/fleet?sslmode=disable` | PostgreSQL connection string |
| `RABBITMQ_URL` | `amqp://guest:guest@localhost:5672/` | RabbitMQ connection URL |
| `MQTT_BROKER` | `tcp://localhost:1883` | MQTT broker address |
| `MQTT_CLIENT_ID` | `fleet-server` | MQTT client identifier |
| `HTTP_PORT` | `8080` | HTTP server port |

## Makefile Commands

| Command | Description |
|---|---|
| `make run` | Run the server |
| `make publisher INTERVAL=2` | Run mock MQTT publisher (interval in seconds) |
| `make event-listener` | Run RabbitMQ geofence alert consumer |
| `make test` | Run unit tests |
| `make lint` | Run golangci-lint |
| `make fmt` | Run gofmt |
| `make infra` | Start infrastructure (Docker Compose) |
| `make infra-down` | Stop infrastructure and remove volumes |
| `make integration-test` | Run integration test script |
| `make build` | Build server binary |
