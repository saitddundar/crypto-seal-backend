# Crypto Seal Backend

A microservices-based backend system for cryptographic document sealing and verification. This project demonstrates modern software architecture principles using Go and Docker.

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                         FRONTEND                                │
│                    (External Client)                            │
└──────────────────────────┬──────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────────┐
│                    DOCKER NETWORK                               │
│                  (crypto-seal-network)                          │
│                                                                 │
│   ┌─────────────────────┐       ┌─────────────────────────┐    │
│   │   NOTARY SERVICE    │──────▶│    HASHER SERVICE       │    │
│   │                     │       │                         │    │
│   │   Port: 8082        │       │   Port: 8081            │    │
│   │                     │       │                         │    │
│   │   Responsibilities: │       │   Responsibilities:     │    │
│   │   - Seal documents  │       │   - SHA256 hashing      │    │
│   │   - Verify seals    │       │   - Stateless compute   │    │
│   │   - Resolve hashes  │       │                         │    │
│   │   - Store records   │       │                         │    │
│   └─────────────────────┘       └─────────────────────────┘    │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

## Microservices Design

This project follows the **Microservices Architecture** pattern, where the application is structured as a collection of loosely coupled, independently deployable services.

### Service Breakdown

| Service | Port | Responsibility | Communication |
|---------|------|----------------|---------------|
| **Hasher Service** | 8081 | Cryptographic hashing (SHA256) | REST API |
| **Notary Service** | 8082 | Document sealing, verification, storage | REST API → Hasher Service |

### Design Principles

1. **Single Responsibility Principle (SRP)**
   - Each service has one specific job
   - Hasher Service: Only computes hashes
   - Notary Service: Only manages seal records

2. **Loose Coupling**
   - Services communicate via HTTP REST APIs
   - No direct dependencies on internal implementations
   - Environment-based service discovery

3. **High Cohesion**
   - Related functionality grouped within the same service
   - Clear boundaries between domains

4. **Stateless Services** (Hasher)
   - Hasher Service is completely stateless
   - Enables horizontal scaling without session management

5. **Service Isolation**
   - Each service runs in its own Docker container
   - Independent failure domains
   - Can be developed, deployed, and scaled independently

## Technology Stack

| Component | Technology |
|-----------|------------|
| Language | Go 1.21 |
| Containerization | Docker |
| Orchestration | Docker Compose |
| Protocol | REST/HTTP |
| Hashing Algorithm | SHA-256 |

## API Endpoints

### Hasher Service (`:8081`)

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/hash` | Compute SHA256 hash of text |
| `GET` | `/health` | Health check |

### Notary Service (`:8082`)

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/seal` | Seal a document (hash + timestamp) |
| `POST` | `/verify` | Verify if a document was sealed |
| `POST` | `/resolve` | Find seal record by hash |
| `GET` | `/list` | List all sealed documents |
| `GET` | `/health` | Health check |

## Getting Started

### Prerequisites

- Docker
- Docker Compose

### Running the Services

```bash
# Start all services
docker-compose up --build

# Run in detached mode
docker-compose up -d --build

# Stop all services
docker-compose down
```

### Testing the API

**Seal a document:**
```bash
curl -X POST http://localhost:8082/seal \
  -H "Content-Type: application/json" \
  -d '{"text": "My important document"}'
```

**Verify a document:**
```bash
curl -X POST http://localhost:8082/verify \
  -H "Content-Type: application/json" \
  -d '{"text": "My important document"}'
```

**Direct hash computation:**
```bash
curl -X POST http://localhost:8081/hash \
  -H "Content-Type: application/json" \
  -d '{"text": "Hello World"}'
```

## Project Structure

```
crypto-seal-backend/
├── docker-compose.yml          # Service orchestration
├── hasher-service/
│   ├── Dockerfile              # Multi-stage build
│   ├── go.mod                  # Go module definition
│   └── main.go                 # Hasher service implementation
├── notary-service/
│   ├── Dockerfile              # Multi-stage build
│   ├── go.mod                  # Go module definition
│   └── main.go                 # Notary service implementation
└── README.md                   # This file
```

## Inter-Service Communication

The Notary Service communicates with the Hasher Service using synchronous HTTP calls:

1. Client sends text to Notary Service
2. Notary Service forwards text to Hasher Service
3. Hasher Service computes SHA256 and returns hash
4. Notary Service stores the hash with metadata
5. Response is sent back to the client

This pattern ensures:
- **Separation of Concerns**: Hashing logic is isolated
- **Reusability**: Hasher can be used by other services
- **Testability**: Each service can be tested independently

## Health Checks & Resilience

- Both services expose `/health` endpoints
- Docker Compose configured with health checks
- Notary Service waits for Hasher Service to be healthy before starting
- Automatic restart policy: `unless-stopped`

## Future Improvements

- [ ] Add persistent storage (PostgreSQL/Redis)
- [ ] Implement API Gateway
- [ ] Add authentication/authorization
- [ ] Implement message queue for async processing
- [ ] Add Kubernetes deployment manifests
- [ ] Implement distributed tracing (Jaeger/Zipkin)

## License

This project is open source and available under the MIT License.
