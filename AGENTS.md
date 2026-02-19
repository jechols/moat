# AGENTS.md

## Project Overview
**MOAT** (Mock Orcid API Thing) is a lightweight mock service for the ORCID API v3.0. It allows developers to test authentication flows and basic record operations (Works, Employment) without relying on the external ORCID sandbox.

## Environment & Setup
- **Language**: Go
- **Go Version Warning**: `go.mod` specifies `go 1.25.3`, but the environment may be running an older version (e.g., 1.24.x). This may cause LSP errors but standard library code should largely remain compatible.
- **Dependencies**: Standard library only (no external modules).

## Essential Commands

### Build & Run
The project is contained within a single file.

```bash
# Run directly
go run main.go

# Build and run
go build -o moat main.go && ./moat
```

Server starts on port **:8080** by default. Use `MOAT_PORT` or `PORT` environment variables to override.

```bash
# Run on port 9090
MOAT_PORT=9090 go run main.go
```

### Testing (Manual)
There are no automated tests (`*_test.go`). Verify functionality using `curl`:

```bash
# Get OAuth Token
curl -X POST http://localhost:8080/oauth/token -d 'client_id=APP-123&grant_type=client_credentials'

# Get Record
curl http://localhost:8080/v3.0/0000-0001-2345-6789/record

# Search
curl "http://localhost:8080/v3.0/search?q=test"
```

## Code Structure
- **`main.go`**: Contains the entire application logic.
  - **Models**: simplified Go structs mirroring ORCID v3 JSON format.
  - **Store**: Global in-memory `dataStore` (reset on restart).
  - **Handlers**: specific functions for Token, Record, Work, Employment, and Search endpoints.
  - **Middleware**: Simple logging and content-type middleware.

## API Surface
Mocked endpoints (prefix: `http://localhost:8080`):
- `POST /oauth/token` - Returns static mock token.
- `GET /v3.0/{orcid}/record` - Returns hardcoded full profile.
- `GET /v3.0/search` - returns static search results.
- `GET/POST/PUT /v3.0/{orcid}/work/*` - Mock work operations.
- `GET/POST/PUT /v3.0/{orcid}/employment/*` - Mock employment operations.

## Gotchas & Limitations
1. **Data Persistence**: Data is in-memory only and resets on restart. Most write operations (POST/PUT) return success but **do not actually modify** the returned read models in this implementation (they are mocked/static).
2. **Logic Shortcuts**:
   - `put-code` generation is random.
   - Search logic is extremely basic (returns 1 result unless query contains "error").
3. **Configuration**: Port is configurable via `MOAT_PORT` (or `PORT`), defaulting to `:8080`.
4. **Go Version**: If you encounter "requires go >= 1.25.3" errors, you may need to edit `go.mod` to match the local Go version or ignore the warning if compilation succeeds.

## Development Patterns
- **No External Libs**: Keep it that way to ensure easy portability.
- **Simplicity First**: This is a mock server; complex business logic isn't required, just correct API contract adherence.
