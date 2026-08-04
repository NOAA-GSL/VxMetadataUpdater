# meta_update_middleware Testing Guide

This guide describes how to validate behavior in this package after code or SQL template changes.

## Run All Tests

From repository root [vxMetadataUpdater](.):

```bash
go test ./...
```

If you see failures only in full-suite runs, disable test cache and show verbose output:

```bash
go test -count=1 -v ./...
```

## Run Specific Test Groups

### Config And Credentials Parsing

```bash
go test -run 'TestParseConfig|TestGetCredentials'
```

Files:

- [meta_update_middleware/meta_update_test.go](meta_update_middleware/meta_update_test.go)

### Query Profiling Helpers

```bash
go test -run 'TestSetQueryProfilingOptions|TestNewQueryOptions|TestRecordQuerySummary|TestPrintQueryProfilingSummary|TestConfigureCapellaTLSOptions'
```

Files:

- [meta_update_middleware/db_utils_test.go](meta_update_middleware/db_utils_test.go)

### SQL Template Validation

```bash
go test -run 'TestSQLTemplate'
```

Files:

- [meta_update_middleware/sql_templates_test.go](meta_update_middleware/sql_templates_test.go)

### Metadata File Writing

```bash
go test -run 'TestWriteStructToFile|TestWriteMetadata'
```

Files:

- [meta_update_middleware/write_to_db_test.go](meta_update_middleware/write_to_db_test.go)

### Utility Helpers

```bash
go test -run 'TestGetTabbedString|TestJsonPrettyPrint|TestConvertSlice'
```

Files:

- [meta_update_middleware/utils_test.go](meta_update_middleware/utils_test.go)

## Notes On The Malformed JSON Test

The malformed JSON behavior test runs parseConfig in a subprocess because parseConfig currently uses fatal logging for decode errors.

- Test name: `TestParseConfig_MalformedJSON_Exits`
- Location: [meta_update_middleware/meta_update_test.go](meta_update_middleware/meta_update_test.go)

This verifies the current behavior (non-zero process exit) without terminating the main test process.

## When To Run Which Tests

- Changed SQL templates: run SQL template tests first, then full suite.
- Changed query profiling logic: run db-utils tests, then full suite.
- Changed output metadata shape or writing logic: run write-to-db tests, then full suite.
- Changed startup/config code: run config tests, then full suite.

## CI Recommendation

For pull requests touching this package, run at minimum:

```bash
go test ./...
```

## Troubleshooting Full-Suite Failures

When individual tests pass but `go test ./...` fails:

- Confirm you are in repo root before running tests.
- Re-run uncached: `go test -count=1 ./...`
- Re-run with randomized order to expose hidden state coupling: `go test -count=1 -shuffle=on ./...`
- Re-run with race detector: `go test -count=1 -race ./...`
- Capella TLS tests now isolate `CACERT_REQUIRED` and `CACERT_FILE` internally. If testing an older branch, clear those shell variables before running the suite.

If it still fails, capture and share the first failing block:

```bash
go test -count=1 -v ./... 2>&1 | tee /tmp/vxmetadataupdater-tests.log
```
