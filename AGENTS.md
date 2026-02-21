# AGENTS.md

## Project Overview

**MOAT** (Mock Orcid API Thing) is a lightweight mock service for the ORCID API
v3.0. It allows developers to test authentication flows and basic record
operations (Works, Employment) without relying on the external ORCID sandbox.

## Environment & Setup

- **Language**: Go
- **Go Version Note**: `go.mod` must not be changed. Tests will run just fine
  with whatever version is installed.
- **Dependencies**: Standard library only (no external modules).

## Essential Commands

### Build & Run

The project is contained within a single file.

```bash
# Build and run
make && ./bin/moat
```

Server starts on port **:8080** by default. Use `MOAT_PORT` or `PORT`
environment variables to override.

```bash
# Run on port 9090
MOAT_PORT=9090 ./bin/moat
```

### Testing

#### Automated Tests

The project includes unit tests covering core functionality. Run them with:

```bash
make test
```

#### Manual Verification

Do **not** use `curl` for manual testing (it is restricted). Instead, use the
`fetch` tool to verify accessible endpoints.

- **Get Record**: Fetch `http://localhost:8080/v3.0/0000-0001-2345-6789/record`
- **Search**: Fetch `http://localhost:8080/v3.0/search?q=test`

For POST endpoints (like `/oauth/token`), use the unit tests or a temporary Go
script to verify behavior.

## Code Structure

- **`main.go`**: Contains the entire application logic.
  - **Models**: simplified Go structs mirroring ORCID v3 JSON format.
  - **Store**: Global in-memory `dataStore` (reset on restart).
  - **Handlers**: specific functions for Token, Record, Work, Employment, and Search endpoints.
  - **Middleware**: Simple logging and content-type middleware.

## API Surface

Mocked endpoints (prefix: `http://localhost:8080`):
- `POST /oauth/token` - Returns static mock token. (Always JSON)
- `GET /v3.0/{orcid}/record` - Returns hardcoded full profile.
- `GET /v3.0/search` - returns static search results.
- `GET/POST/PUT /v3.0/{orcid}/work/*` - Mock work operations.
- `GET/POST/PUT /v3.0/{orcid}/employment/*` - Mock employment operations.

**Note**: All `/v3.0/*` endpoints default to **XML** responses unless `Accept: application/json` header is present. This mimics the real ORCID API behavior.

## Gotchas & Limitations

1. **Data Persistence**: Data is in-memory only and resets on restart. Most
   write operations (POST/PUT) return success but **do not actually modify**
   the returned read models in this implementation (they are mocked/static).
2. **Logic Shortcuts**:
   - `put-code` generation is random.
   - Search logic is extremely basic (returns 1 result unless query contains
     "error").
3. **Configuration**: Port is configurable via `MOAT_PORT` (or `PORT`),
   defaulting to `:8080`.

## Development Patterns

- **No External Libs**: Keep it that way to ensure easy portability.
- **Simplicity First**: This is a mock server; complex business logic isn't
  required, just correct API contract adherence.
