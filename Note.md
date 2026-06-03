# Paygate Build Resolution & Service Notes


## What I Removed and Why

### 1. Azure SDK (`azure-pipeline-go`, `azure-storage-blob-go`, all `go-autorest/*`)

**Removed from:** `go.mod`, `pkg/transfers/pipeline/audittrail/storage_blob.go`

The Azure SDK for Go underwent a monorepo-to-split-modules migration. This left two versions
of the same packages simultaneously resolvable by Go's module graph:

- `github.com/Azure/go-autorest v12.0.0+incompatible` (old monorepo)
- `github.com/Azure/go-autorest/autorest/adal v0.8.3` (new split module)

Go cannot pick between them and throws an `ambiguous import` error that blocks `go mod tidy`
entirely. Since UltraPay does not use Azure Blob Storage, removing the `azureblob` blank import
from `storage_blob.go` killed the entire dependency chain.

---

### 2. `gocloud.dev/blob/gcsblob` and `gocloud.dev/blob/s3blob`

**Removed from:** `pkg/transfers/pipeline/audittrail/storage_blob.go`

Both drivers share `gocloud.dev/internal/testing/setup` in their test suites. That internal
test package imports `azure-storage-blob-go` which re-introduces the same Azure ambiguous
import error through the test dependency graph. Only `fileblob` and `memblob` were kept —
neither has cloud SDK test dependencies.

---

### 3. `gocloud.dev/pubsub/kafkapubsub`

**Removed from:** `go.mod`, `pkg/stream/stream.go`, `pkg/transfers/pipeline/publisher_kafka.go`,
`pkg/transfers/pipeline/subscription.go`

Its test suite imports the same `gocloud.dev/internal/testing/setup` → Azure chain.
The only thing `kafkapubsub` provided was `MinimalConfig()` — a two-line function that returns
a `*sarama.Config`:

```go
config := sarama.NewConfig()
config.Version = sarama.V0_11_0_0
```

This was inlined directly into `publisher_kafka.go` and `subscription.go` using the existing
`github.com/Shopify/sarama` dependency. No functionality was lost.

---

### 4. `gocloud.dev/secrets/hashivault`

**Removed from:** `go.mod`

Indirect dependency. Not referenced anywhere in the paygate codebase or config. Dropped cleanly.

---

### 5. `mattn/go-sqlite3` → replaced with `modernc.org/sqlite`

**Changed in:** `go.mod`, `pkg/database/sqlite.go`

`mattn/go-sqlite3` requires CGO — it wraps a C library and needs `gcc` to compile.
On Windows without TDM-GCC this produces:

```
cgo: C compiler "gcc" not found: executable file not found in %PATH%
```

`modernc.org/sqlite` is a pure-Go SQLite implementation transpiled from the original C source.
It is fully compatible at the SQL and `database/sql` interface level. Three changes were made
to `sqlite.go`:

| Change | Old | New |
|--------|-----|-----|
| Driver name | `sql.Open("sqlite3", ...)` | `sql.Open("sqlite", ...)` |
| Version log | `sqlite3.Version()` CGO call | Static log string |
| Unique violation check | `sqlite3.Error` type assertion + string check | String check only (modernc returns identical error message) |

---

### 6. `honnef.co/go/tools` and `github.com/PuerkitoBio/goquery`

**Removed from:** `go.mod`

`honnef.co/go/tools` is a static analysis / linter package (staticcheck) — dev tooling with
no runtime relevance. `goquery` is an HTML scraper pulled in transitively. Neither has any
role in payment infrastructure.

---

## Re-adding Removed Storage Drivers (Production Guidance)

### Azure Blob Storage

Do **not** attempt to re-add `gocloud.dev/blob/azureblob` on `gocloud.dev v0.20.0`.
The Azure SDK ambiguity is unresolvable at that version.

To use Azure Blob Storage in production, upgrade gocloud.dev to `v0.26.0` or later
(where the issue was resolved upstream) and add the modern split sub-modules:

```go
github.com/Azure/go-autorest/autorest v0.11.x
github.com/Azure/go-autorest/autorest/adal v0.9.x
github.com/Azure/go-autorest/autorest/azure/auth v0.5.x
```

Then restore the import in `storage_blob.go`:

```go
_ "gocloud.dev/blob/azureblob"
```

### GCS Blob Storage

Same gocloud.dev version constraint applies. Upgrade to `v0.26.0+` first, then restore:

```go
_ "gocloud.dev/blob/gcsblob"
```

### S3 Blob Storage (Most Likely for Production)

S3 is the recommended audit trail storage backend for production ACH infrastructure.
Same upgrade path — gocloud.dev `v0.26.0+` — then restore:

```go
_ "gocloud.dev/blob/s3blob"
```

And update `examples/config.yaml` audit trail section:

```yaml
pipeline:
  auditTrail:
    bucketURI: "s3://your-bucket-name?region=us-east-1"
```

---

## FTP Container Issue

The server logs this warning on startup:

```
problem with upload.Agent connection: ftp: ftp:2121 is not whitelisted: unable to resolve (found 0) ftp: lookup ftp: no such host
```

This is **non-fatal** — paygate continues running. Two issues explain it:

**Issue 1 — Hostname not resolvable natively:**
The hostname `ftp` only resolves inside a Docker network where a container is explicitly
named `ftp`. Running paygate natively on Windows, your OS DNS has no record for `ftp`
as a hostname.

**Issue 2 — Whitelist:**
Paygate maintains an internal allowlist of permitted FTP/SFTP hostnames. The bare hostname
`ftp` is not on it by default.

**Fix for Docker Compose deployment:**
Ensure your `docker-compose.yml` names the FTP service container `ftp` and that paygate
runs in the same Docker network. Docker's internal DNS will then resolve `ftp` correctly
and the config requires no changes.

**Fix for native Windows development (no Docker):**
Replace `ftp:2121` in `examples/config.yaml` with `localhost:2121` and run a local FTP
server on that port, or switch to local filesystem storage entirely:

```yaml
odfi:
  ftp:
    hostname: "localhost:2121"
    username: "admin"
    password: "123456"
```

---

## Service Endpoints

### Externally Accessible (confirmed working)

| Endpoint | Purpose |
|----------|---------|
| `http://localhost:8082` | **Paygate Transfers API** — create and manage ACH transfers, micro-deposits, organizations |
| `http://localhost:9092` | **Paygate Admin API** — Prometheus metrics, health checks, internal operational controls |

**Quick verification:**
```bash
curl http://localhost:8082/ping    # returns: PONG
curl http://localhost:9092/metrics # returns: Prometheus metrics output
```

### Internal Service-to-Service (Docker network only)

| Endpoint | Purpose |
|----------|---------|
| `http://customers:8087` | **moov-io/customers** — customer identity and bank account management |

`http://customers:8087` is **not** a localhost address. It is a Docker internal hostname
that only resolves when both paygate and the customers service are running in the same
Docker Compose network. Hitting it directly from a browser or curl on Windows will fail
with a DNS error — this is correct and expected behaviour.

Paygate calls this endpoint internally when processing transfers to validate the source
and destination customer accounts. You interact with the customers service directly only
for setup operations: creating customer records and linking bank accounts before
initiating transfers through paygate.

`localhost:8082` and `localhost:9092` are the **only two externally accessible endpoints**
from this paygate instance.

---

