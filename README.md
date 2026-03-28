# Card Client

## Architecture Overview

```
┌─────────────────────────────────┐
│     Client/Presentation Layer    │
│         ServiceManager           │
└──────────────┬──────────────────┘
               │
┌──────────────▼──────────────────┐
│         Facade Layer            │
│  Single API for all operations  │
└──────────────┬──────────────────┘
               │
        ┌──────┴──────┬──────────┐
        │             │          │
┌───────▼──────┐  ┌───▼────┐  ┌──▼───┐
│   Catalog    │  │ Tracker│  │Router│
│ (Data Layer) │  │        │  │      │
└───────┬──────┘  └────────┘  └──────┘
        │
   ┌────┴────────────┬───────────┐
   │                 │           │
┌──▼───────────┐  ┌──▼────────┐  │
│Record Store  │  │Blob Store │  │
│  (SQLite)    │  │ (Pebble)  │  │
└──────────────┘  └───────────┘  │
```

## Key Components

### Core Services

- **Facade** (`/facade`) - High-level API that orchestrates all operations
    - VaultManager - Manages vault lifecycle with RWMutex protection
    - QueryService - Read operations
    - SyncService - Import/update operations
    - ExportService - Card export with templating
    - FavoriteService - Favorite management
    - CacheManager - Operation result caching

- **Catalog** (`/store/catalog`) - Unified interface for data operations
    - Wraps Record Store (metadata) and Blob Store (binary files)
    - Ensures transactional consistency between stores

- **Dual Storage Strategy**
    - **SQLite** (`/store/record/erecord`) - Structured metadata with Ent ORM
    - **Pebble KV** (`/store/blob/pblob`) - Binary card files with versioning

## Project Structure

```
card-client/
├── client/          # Application initialization
├── facade/          # High-level API facade
├── store/           # Data access layer
│   ├── resource/    # Domain models (Record, Filter, etc.)
│   ├── catalog/     # Unified store interface
│   ├── record/      # Record storage abstraction
│   └── blob/        # Binary storage abstraction
├── library/         # Multi-vault management
├── preferences/     # Configuration (Viper-based)
├── credentials/     # Secure credential handling (keyring)
├── operation/       # Async operation tracking
├── tracker/         # Concurrency control
└── generate/        # Code generation tools
```

## Stack

- **Go 1.26.0** - Core language
- **Ent ORM** (v0.14.5) - SQL schema generation and queries
- **SQLite** - Record metadata storage
- **Pebble** (v2.1.4) - High-performance KV store for binary files
- **Viper** - Configuration management
- **ZeroLog** - Structured logging

## Architecture Highlights

### 1. Hybrid Storage Design

- SQLite for queryable metadata (fast searches, transactions)
- Pebble KV for binary files (efficient versioning, fast access)
- Prevents large blobs from impacting query performance

### 2. Concurrency Model

- **Vault-level RWMutex**: Exclusive writes, concurrent reads
- **Per-card mutex tracking**: Fine-grained locking for updates
- **Context propagation**: Cancellable long-running operations
- **Async operations**: Goroutines for imports/exports

### 3. Vault System

- Multiple isolated vaults (separate databases)
- Lazy loading (only when accessed)
- Hot-swappable without restart
- Each vault has its own SQLite + Pebble databases

### 4. Version Tracking

- Keeps multiple versions of each card (default: 5)
- Composite keys: RID (Record ID) + timestamp
- Enables history and rollback capabilities

### 5. Operation Tracking

- In-memory registry of ongoing operations
- Progress reporting with success/failure metrics
- Context-based cancellation support
- Nano ID generation with UUID fallback

## Important Gotchas

### 1. Code Generation

- **Ent ORM generates code** - Don't edit files in `/store/record/erecord/ent/`
- **Post-processing required** - Custom ID types (RID, CID) need manual fixes
- Run `go generate ./generate` after schema changes

### 2. Transaction Handling

- **Nested transactions** - Catalog coordinates Record + Blob transactions
- **Context propagation** - Always pass context for proper transaction scope
- **Defer cleanup** - Use defer for mutex unlocks and transaction rollbacks

### 3. Concurrency Pitfalls

- **Always lock vault** before operations (RLock for reads, Lock for writes)
- **Use tracker** for per-card operations to prevent race conditions
- **Defer unlocks** to ensure cleanup even on panics
- **One vault active** at a time per ServiceManager

### 4. Storage Limitations

- **SQLite connection pool** - Configure max connections carefully
- **Pebble batch size** - Large imports may need batching
- **Version retention** - Old versions auto-pruned (keep max 5)
- **FTS indexing** - Full-text search requires explicit UpsertFTS calls

### 5. Error Handling

- **Check integrity** - Use Record.Integrity() to validate completeness
- **Sync status** - Track SyncStatus enum (unchanged/updated/failed)
- **Operation reports** - Always check success/failure counts
- **Context cancellation** - Handle context.Canceled gracefully

### 6. Performance Considerations

- **Lazy loading** - Vaults only loaded when needed
- **Batch operations** - Use transactions for bulk inserts
- **Index usage** - SQLite indexes on source, URLs, timestamps
- **Thumbnail generation** - On-the-fly from PNG content (may be slow)

### 7. Security Notes

- **Credentials** - Stored in system keyring, not in config files
- **Path traversal** - Validate export paths to prevent directory escape

## Development Tips

### Adding New Storage Backend

1. Implement `record.Builder` or `blob.Builder` interface
2. Register in library's storage maps
3. Update Manifest to support new storage type

### Adding New Facade Service

1. Create service struct in `/facade`
2. Add to Facade composite struct
3. Wire in ServiceManager initialization

### Debugging Concurrency Issues

- Enable debug logging for mutex operations
- Use race detector: `go run -race`
- Check operation registry for stuck operations
- Monitor goroutine count for leaks

### Testing Considerations

- Each test can use separate vault
- Use `enttest` package for test database setup
- Mock external dependencies (router, fetcher)
- Clean up test vaults to prevent disk bloat

## Common Operations Flow

### Import Cards

```
URLs → Validate → Fetch metadata → Check duplicates →
Insert record → Store PNG → Update FTS → Report
```

### Update Cards

```
Lock card → Fetch latest → Compare versions →
Update if changed → Store new version → Unlock → Report
```

### Query Cards

```
RLock vault → Build filter → SQL query →
Transform results → RUnlock → Return Box[T]
```

## Build & Run

```bash
# Generate Ent code
go generate ./generate

# Set sqlite FTS5 flag
export GOFLAGS="-tags=sqlite_fts5"

# Build
go build ./...

# Run tests
go test ./...

# Run with race detector (development)
go run -race .
```

## Dependencies

See `go.mod` for full dependency list. Key external packages:

- `github.com/digital-foxy/card-fetcher` - Protocol handlers
- `github.com/digital-foxy/card-parser` - PNG parsing
- `github.com/digital-foxy/toolkit` - Shared utilities
- `entgo.io/ent` - ORM framework
- `cockroachdb/pebble` - KV store
