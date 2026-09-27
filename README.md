# Royal Move

Royal Move is a polished, self-hosted chess analysis application. It imports recent public games from Chess.com and Lichess or accepts pasted PGN notation, then evaluates every move with a local Stockfish engine.

## Highlights

- Chess.com and Lichess username import
- Full and partial PGN validation
- Per-move evaluation, principal variation, loss, and classification
- Local Stockfish 19 analysis with configurable CPU and memory limits
- Secure account registration and session-based analysis history
- Administrator bootstrap and system overview
- Persistent embedded SQLite storage
- Responsive, accessible classic editorial interface
- No Docker, PostgreSQL, Redis, or cloud account required

## Architecture

```text
Next.js web application (:3000)
              |
              v
        Go REST API (:8080)
         |-- SQLite database
         |-- in-process job queue
         |-- Chess.com adapter
         |-- Lichess adapter
         `-- Stockfish 19 process
```

## Requirements

- Windows 10/11
- Go 1.25+
- Node.js 24+
- Internet access for the first Stockfish download and platform imports

## Quick start

```powershell
git clone <repository-url>
cd chess-analyzer
.\start-local.ps1
```

Open [http://localhost:3000](http://localhost:3000). The setup script installs frontend dependencies when needed, downloads Stockfish from its official GitHub release, builds the Go API, and starts both local processes.

Stop the application with:

```powershell
.\stop-local.ps1
```

Runtime logs are stored under `.runtime/`. Application data is stored in `backend/royal_move.db` and is intentionally excluded from version control.

## Configuration

All configuration is provided through environment variables. See `.env.example` for the complete list.

| Variable | Default | Purpose |
| --- | --- | --- |
| `APP_ENV` | `development` | Runtime environment; secure production checks apply to `production` |
| `API_PORT` | `8080` | Go API port |
| `DATABASE_URL` | local SQLite file | SQLite connection string |
| `JWT_SECRET` | development-only value | Token signing secret; 32+ secure characters required in production |
| `WEB_ORIGIN` | `http://localhost:3000` | Allowed browser origin |
| `ADMIN_EMAIL` | empty | Account email promoted to administrator at registration |
| `STOCKFISH_PATH` | local bundled path | Stockfish executable |
| `ANALYSIS_THREADS` | `2` | CPU threads per Stockfish process |
| `ANALYSIS_HASH_MB` | `256` | Hash memory per Stockfish process |
| `ANALYSIS_WORKERS` | `1` | Concurrent analysis workers |
| `ANALYSIS_QUEUE_CAPACITY` | `256` | Maximum queued jobs |
| `ANALYSIS_MOVETIME_MS` | `800` | Engine time per position |

For production, use a unique `JWT_SECRET`, HTTPS, a persistent backup strategy for SQLite, and a reverse proxy with request and connection limits.

## Development

Backend:

```powershell
cd backend
go test ./...
go run ./cmd/api
```

Frontend:

```powershell
cd web
npm install
npm run dev
npm run lint
npm run typecheck
npm run build
```

Formatting:

```powershell
cd backend
gofmt -w .

cd ..\web
npm run format
```

## API

The OpenAPI description is available at [`docs/openapi.yaml`](docs/openapi.yaml). Main endpoints:

- `POST /api/auth/register`
- `POST /api/auth/login`
- `GET /api/me`
- `GET /api/platforms/{source}/{username}/games`
- `POST /api/analyses`
- `GET /api/analyses`
- `GET /api/analyses/{id}`
- `POST /api/analyses/{id}/retry`
- `GET /health`

## Security

- Passwords are hashed with bcrypt and never returned by the API.
- JWTs expire after seven days and are signed with HS256.
- API responses include security, origin, and rate-limit headers.
- Request bodies are size-limited and reject unknown or trailing JSON fields.
- Users can only access their own analyses; administrators can inspect all jobs.
- Runtime databases, logs, downloaded binaries, and secrets are ignored by Git.

## Third-party services and licensing

Royal Move uses public read-only game data from Chess.com and Lichess. Their names identify the selected source; this project does not copy either platform's visual identity.

Stockfish is licensed under GPLv3. The setup script downloads the unmodified official binary. See [`THIRD_PARTY_NOTICES.md`](THIRD_PARTY_NOTICES.md) for attribution and source information.
