# meta_update_middleware

Builds MATS GUI metadata documents from DD records in Couchbase.

The tool reads app/docType definitions from a settings file, discovers model-level values from DD documents, and writes one consolidated metadata document per app.

This utility depends heavily on having appropriate indexes configured in the database.
There will be a runtime recorded in the output at the end of the run. Excessive runtime
can be an indicator of inproper index configuration.

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

- Go version declared by [meta_update_middleware/go.mod](meta_update_middleware/go.mod)
- Access to a Couchbase bucket/scope/collection that contains DD documents
- Credentials YAML file
- Settings JSON file

## Build And Run

From [meta_update_middleware](meta_update_middleware):

```bash
go build .
./meta-update
```

Or run directly:

```bash
go run .
```

## Docker

For container build and runtime instructions, including bind-mounted credentials, see [README.docker.md](README.docker.md).

## CI Workflow

This repository uses GitHub Actions workflow [ci.yml](.github/workflows/ci.yml).


### Triggers

- Push to any branch
- Push tags matching release patterns:
  - `<major>.<minor>.<patch>`
  - `<major>.<minor>.<patch>-rc<number>`
- Pull requests
- Manual runs (`workflow_dispatch`)

### Expected Behavior

- Commit and push to a feature branch:
  - The CI workflow should run on that branch push.
  - Jobs should execute in sequence: `lint`, `test`, `build-vxmetadataupdater`, then `scan-vxmetadataupdater`.

- Open or update a pull request targeting `main`:
  - The CI workflow should run for the pull request event.
  - This provides pre-merge signal for linting, tests, image build, and scan steps.

- Merge a feature branch into `main`:
  - The merge commit push to `main` should trigger the workflow again.
  - On default branch pushes, image tagging includes `latest` in addition to branch/SHA tags.

### Pipeline Stages

- `lint`: runs `go vet ./...` and enforces `gofmt`
- `test`: runs `go test ./...`
- `build-vxmetadataupdater`: builds and pushes a multi-arch container image (`linux/amd64`, `linux/arm64`)
- `scan-vxmetadataupdater`: scans the published image with Trivy and uploads SARIF results to GitHub Security

### Container Publishing

Images are published to:

- `ghcr.io/noaa-gsl/vxmetadataupdater`

Generated tags include branch, PR, semver, short SHA, and `latest` on the default branch.

### Security Scanning

- Trivy fails the job on `HIGH` and `CRITICAL` findings in the console scan.
- A SARIF report is always generated and uploaded to the GitHub Security tab.

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
  - when using `-p`, also set `-a` so a single metadata document is selected for output

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
- For Capella clusters, ensure the runtime trust store includes the required CA chain for your endpoint.

```text
-----BEGIN CERTIFICATE-----
MIIDFTCCAf2gAwIBAgIRANLVkgOvtaXiQJi0V6qeNtswDQYJKoZIhvcNAQELBQAw
JDESMBAGA1UECgwJQ291Y2hiYXNlMQ4wDAYDVQQLDAVDbG91ZDAeFw0xOTEyMDYy
MjEyNTlaFw0yOTEyMDYyMzEyNTlaMCQxEjAQBgNVBAoMCUNvdWNoYmFzZTEOMAwG.....
.....
-----END CERTIFICATE-----
```

In the event that the cb_host contains "cloud.couchbase.com" it will be
assumed that the host is a Capella cluster and a certificate will be required
in addition to the user and password.

The certificate can be obtained from the Capella UI under the "Connection" tab.

## Settings File Format

Example from [meta_update_middleware/settings.json](meta_update_middleware/settings.json):

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

1. Write metadata to Couchbase or file (`-p`).
2. Print query profiling summary.

## SQL Templates

The middleware query templates live in [meta_update_middleware/sqls](meta_update_middleware/sqls).

Current templates include:

- `getModels.sql`
- `getModelsNoData.sql`
- `getModelsWithMetadata.sql`
- `getDistinctDataKeys.sql`
- `getDistinctFcstLen.sql`
- `getDistinctRegion.sql`
- `getDistinctDisplayText.sql`
- `getDistinctDisplayCategory.sql`
- `getDistinctDisplayOrder.sql`
- `getMinMaxCountFloor.sql`

Template integrity tests are implemented in [meta_update_middleware/sql_templates_test.go](meta_update_middleware/sql_templates_test.go).

## Testing

Run all package tests from [meta_update_middleware](meta_update_middleware):

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

Run the tagged integration test explicitly:

```bash
go test -v -tags integration -test.fullpath=true -timeout 30s -run ^TestIntegration_UpsertAndGet ./tests/integration
```

Set these environment variables first: `CB_CONN`, `CB_USER`, `CB_PASS`, `CB_BUCKET`, `CB_SCOPE`, `CB_COLLECTION`.

See [meta_update_middleware/TESTING.md](meta_update_middleware/TESTING.md) for a focused test guide.

## Operational Notes

- The executable logs with file and line (`log.Lshortfile`).
- Invalid `-query-profile` values terminate execution.
- If parsing settings JSON fails, current implementation exits via fatal log.
- If `-p` is provided, pair it with `-a` so a single app/docType selection writes to the file path.

## Historical Notes

Legacy performance notes remain in [meta_update_middleware/docs/performance.txt](meta_update_middleware/docs/performance.txt).
