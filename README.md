# meta_update_middleware

Builds MATS GUI metadata documents from DD records in Couchbase.

The tool reads app/docType definitions from a settings file, discovers model-level values from DD documents, and writes one consolidated metadata document per app.

## What It Produces

For each settings entry, this package writes one metadata document with key format:

`MD:matsGui:<name>:COMMON:V01`

Example app names from the default settings:

- `ceiling`
- `visibility`
- `surface`

The generated JSON includes model metadata such as:

- `fcstLens`
- `regions`
- `displayText`
- `displayCategory`
- `displayOrder`
- `mindate`
- `maxdate`
- `numrecs`

For `docType == SUMS`, data keys are written to `variables`.
For other docTypes (for example `CTC`), data keys are written to `thresholds`.

## Requirements

- Go version declared by [go.mod](go.mod)
- Access to a Couchbase bucket/scope/collection that contains DD documents
- Credentials YAML file (must be `chmod 600`; the tool refuses to start if the file is group- or world-readable)
- Settings JSON file

## Build And Run

From the repository root:

```bash
go build .
./meta-update
```

Or run directly:

```bash
go run .
```

## Docker Build And Run

Build the image from the repository root:

```bash
docker build -t vxmetadataupdater:latest .
```

Run with local credentials and settings mounted read-only:

```bash
docker run --rm \
  -v "$HOME/credentials:/run/secrets/credentials:ro" \
  -v "$PWD/settings.json:/app/settings.json:ro" \
  vxmetadataupdater:latest \
  -c /run/secrets/credentials \
  -s /app/settings.json
```

Run for a single app:

```bash
docker run --rm \
  -v "$HOME/credentials:/run/secrets/credentials:ro" \
  -v "$PWD/settings.json:/app/settings.json:ro" \
  vxmetadataupdater:latest \
  -c /run/secrets/credentials \
  -s /app/settings.json \
  -a ceiling
```

Write output JSON to the host instead of Couchbase:

```bash
docker run --rm \
  -v "$HOME/credentials:/run/secrets/credentials:ro" \
  -v "$PWD/settings.json:/app/settings.json:ro" \
  -v "$PWD/out:/out" \
  vxmetadataupdater:latest \
  -c /run/secrets/credentials \
  -s /app/settings.json \
  -a ceiling \
  -p /out/metadata_ceiling.json
```

For Capella, pass required certificate environment and mount the certificate file:

```bash
docker run --rm \
  -e CACERT_REQUIRED=1 \
  -e CACERT_FILE=/certs/capella-ca.pem \
  -v "$PWD/capella-ca.pem:/certs/capella-ca.pem:ro" \
  -v "$HOME/credentials:/run/secrets/credentials:ro" \
  -v "$PWD/settings.json:/app/settings.json:ro" \
  vxmetadataupdater:latest \
  -c /run/secrets/credentials \
  -s /app/settings.json
```

Notes:

- The container runs as a non-root user.
- Keep the credentials file permissions at `600` on the host; the application validates this.

## CLI Flags

### Core Input Flags

- `-c`: path to credentials YAML
  - default: `$HOME/credentials`
- `-s`: path to settings JSON
  - default: `./settings.json`
- `-a`: app name filter
  - default: empty (process all apps in settings)
- `-p`: write output metadata JSON to a file path instead of writing to Couchbase
  - default: empty (write to Couchbase)

### Query Profiling Flags

- `-query-metrics`: enable Couchbase query metrics
  - default: `true`
- `-query-profile`: profiling mode (`off`, `phases`, `timings`)
  - default: `off`
- `-query-slow-ms`: only log detailed query metadata when elapsed time is at least this threshold
  - default: `500`
- `-query-summary-top`: number of slow query templates included in the end-of-run summary
  - use `0` to show all
  - default: `10`

### Runtime Profiling Flags

- `-cpuprofile`: write CPU profile to file
- `-memprofile`: write heap profile to file

Detailed profiling guidance is in [meta_update_middleware/PROFILING.md](meta_update_middleware/PROFILING.md).

## Credentials File Format

Example:

```yaml
cb_host: couchbase://adb-cb1.example.org
cb_user: my_user
cb_password: my_password
cb_bucket: vxdata
cb_scope: _default
cb_collection: METAR
cb_timeout_seconds: 3600
```

Notes:

- If `cb_timeout_seconds` is omitted or `0`, the tool uses `3600` seconds for query timeout.
- For multi-node targets, use a Couchbase connection string accepted by the Go SDK.
- The file must be readable only by its owner (`chmod 600`). The tool fatals on startup if group or other read bits are set.

### Capella Clusters

When `cb_host` contains `cloud.couchbase.com`, the tool treats the cluster as Capella and requires two additional environment variables:

| Variable          | Description                                                                                   |
| ----------------- | --------------------------------------------------------------------------------------------- |
| `CACERT_REQUIRED` | Set to any non-empty value to enable TLS certificate validation                               |
| `CACERT_FILE`     | Path to the CA certificate PEM file downloaded from the Capella UI (Connection → Certificate) |

Example certificate file format:

```text
-----BEGIN CERTIFICATE-----
MIIDFTCCAf2gAwIBAgIRANLVkgOvtaXiQJi0V6qeNtswDQYJKoZIhvcNAQELBQAw
JDESMBAGA1UECgwJQ291Y2hiYXNlMQ4wDAYDVQQLDAVDbG91ZDAeFw0xOTEyMDYy
MjEyNTlaFw0yOTEyMDYyMzEyNTlaMCQxEjAQBgNVBAoMCUNvdWNoYmFzZTEOMAwG
.....
-----END CERTIFICATE-----
```

## Settings File Format

Example from [settings.json](settings.json):

```json
{
 "metadata": [
  {
   "name": "ceiling",
   "app": "cb-ceiling",
   "docType": ["CTC"],
   "subDocType": "CEILING"
  }
 ]
}
```

Fields:

- `name`: used in generated metadata document key
- `app`: app label stored in metadata
- `docType`: array of docTypes to process for that app
- `subDocType`: DD subDocType filter

## Common Commands

Run for all apps in settings:

```bash
go run . -c ~/credentials -s ./settings.json
```

Run for one app:

```bash
go run . -c ~/credentials -s ./settings.json -a ceiling
```

Write output JSON to a local file (no DB write):

```bash
go run . -c ~/credentials -s ./settings.json -a ceiling -p ./metadata_ceiling.json
```

Run with query and runtime profiling enabled:

```bash
go run . -c ~/credentials -s ./settings.json -a ceiling \
 -query-profile=timings -query-slow-ms=0 -query-summary-top=20 \
 -cpuprofile cpu.pprof -memprofile mem.pprof
```

## Data Flow Summary

1. Read settings and credentials.
2. Open Couchbase connection.
3. For each selected app/docType:

   - get models requiring metadata
   - read data keys, forecast lengths, regions, display fields, and min/max/count stats
   - assemble one `MetadataJSON` document with `models[]`

4. Write metadata to Couchbase or file (`-p`).
5. Print query profiling summary.

## SQL Templates

Query templates live in [sqls/](sqls/).

| Template                         | Purpose                                           |
| -------------------------------- | ------------------------------------------------- |
| `getModels.sql`                  | Models with DD data that require metadata         |
| `getModelsNoData.sql`            | Models with metadata but no matching DD data      |
| `getModelsWithMetadata.sql`      | Models that already have a metadata document      |
| `getDistinctDataKeys.sql`        | Distinct variable/threshold keys for a model      |
| `getDistinctFcstLen.sql`         | Distinct forecast lengths for a model             |
| `getDistinctRegion.sql`          | Distinct regions for a model                      |
| `getDistinctDisplayText.sql`     | Display text label for a model                    |
| `getDistinctDisplayCategory.sql` | Display category for a model                      |
| `getDistinctDisplayOrder.sql`    | Display order for a model                         |
| `getMinMaxCountFloor.sql`        | Min/max dates, record count, and update timestamp |

Template integrity tests are in [sql_templates_test.go](sql_templates_test.go).

## Testing

Run all package tests:

```bash
go test ./...
```

The test suite covers:

- config and credentials parsing
- malformed JSON parse behavior
- utility helpers
- SQL template placeholder/substitution checks
- query profiling state helpers
- metadata file writing behavior

See [TESTING.md](TESTING.md) for a focused test guide.

## Operational Notes

- The executable logs with file and line (`log.Lshortfile`).
- Invalid `-query-profile` values terminate execution.
- If parsing settings JSON fails, current implementation exits via fatal log.
- If `-p` is provided, metadata is written only to that file path for each processed app/docType iteration.

## Historical Notes

Legacy performance notes remain in [meta_update_middleware/docs/performance.txt](meta_update_middleware/docs/performance.txt).
