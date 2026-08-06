package main

import (
	"crypto/x509"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/couchbase/gocb/v2"
)

// CbConnection bundles the live Couchbase handles needed for queries.
// vxDBTARGET is the N1QL FROM target in "bucket.scope.collection" form.
type CbConnection struct {
	Cluster    *gocb.Cluster
	Bucket     *gocb.Bucket
	Scope      *gocb.Scope
	Collection *gocb.Collection
	vxDBTARGET string
}

var queryProfilingConfig = struct {
	metricsEnabled  bool
	profileMode     gocb.QueryProfileMode
	slowQueryCutoff time.Duration
}{
	metricsEnabled:  true,
	profileMode:     gocb.QueryProfileModeNone,
	slowQueryCutoff: 500 * time.Millisecond,
}

// querySummary accumulates per-template timing statistics for the end-of-run report.
type querySummary struct {
	Tag            string
	QueryText      string
	Count          int
	TotalElapsed   time.Duration
	TotalExecution time.Duration
	MaxElapsed     time.Duration
}

var querySummaryState = struct {
	sync.Mutex
	byKey map[string]*querySummary
}{
	byKey: map[string]*querySummary{},
}

// setQueryProfilingOptions configures the package-level query profiling settings.
// profile must be one of "off", "phases", or "timings"; slowMs < 0 is clamped to 0.
func setQueryProfilingOptions(metrics bool, profile string, slowMs int) {
	queryProfilingConfig.metricsEnabled = metrics

	switch strings.ToLower(strings.TrimSpace(profile)) {
	case "", "off", "none":
		queryProfilingConfig.profileMode = gocb.QueryProfileModeNone
	case "phases":
		queryProfilingConfig.profileMode = gocb.QueryProfileModePhases
	case "timings":
		queryProfilingConfig.profileMode = gocb.QueryProfileModeTimings
	default:
		log.Fatalf("invalid -query-profile value %q, expected off|phases|timings", profile)
	}

	if slowMs < 0 {
		slowMs = 0
	}
	queryProfilingConfig.slowQueryCutoff = time.Duration(slowMs) * time.Millisecond

	log.Printf("query profiling configured: metrics=%t profile=%s slow_query_ms=%d", queryProfilingConfig.metricsEnabled, queryProfilingConfig.profileMode, slowMs)
}

// newQueryOptions returns QueryOptions reflecting the current profiling configuration.
// Adhoc is always true; prepared statements are not used.
func newQueryOptions() *gocb.QueryOptions {
	return &gocb.QueryOptions{
		Adhoc:   true,
		Metrics: queryProfilingConfig.metricsEnabled,
		Profile: queryProfilingConfig.profileMode,
	}
}

// finalizeQueryResult closes a query result, records its timing, and logs a
// detail line when elapsed time meets or exceeds the configured slow-query threshold.
func finalizeQueryResult(tag string, queryText string, start time.Time, queryResult *gocb.QueryResult) {
	if err := queryResult.Err(); err != nil {
		log.Fatal(err)
	}

	meta, err := queryResult.MetaData()
	if err != nil {
		log.Fatal(err)
	}

	elapsed := time.Since(start)
	recordQuerySummary(tag, queryText, elapsed, meta.Metrics.ExecutionTime)
	if queryProfilingConfig.slowQueryCutoff > 0 && elapsed < queryProfilingConfig.slowQueryCutoff {
		return
	}

	if meta.Metrics.ElapsedTime == 0 && meta.Metrics.ExecutionTime == 0 {
		return
	}

	log.Printf("query profile [%s] elapsed=%v execution=%v count=%d warnings=%d status=%s", tag, meta.Metrics.ElapsedTime, meta.Metrics.ExecutionTime, meta.Metrics.ResultCount, len(meta.Warnings), meta.Status)
	log.Printf("query text [%s]: %s", tag, strings.TrimSpace(queryText))

	if queryProfilingConfig.profileMode != gocb.QueryProfileModeNone && meta.Profile != nil {
		profileBytes, marshalErr := json.Marshal(meta.Profile)
		if marshalErr != nil {
			log.Printf("query profile json marshal failed [%s]: %v", tag, marshalErr)
			return
		}
		log.Printf("query profile details [%s]: %s", tag, string(profileBytes))
	}
}

// recordQuerySummary accumulates elapsed/execution timing for a query template.
// Entries are keyed by tag + trimmed query text so whitespace variants are merged.
func recordQuerySummary(tag string, queryText string, elapsed time.Duration, execution time.Duration) {
	key := tag + "|" + strings.TrimSpace(queryText)

	querySummaryState.Lock()
	defer querySummaryState.Unlock()

	s, ok := querySummaryState.byKey[key]
	if !ok {
		s = &querySummary{Tag: tag, QueryText: strings.TrimSpace(queryText)}
		querySummaryState.byKey[key] = s
	}

	s.Count++
	s.TotalElapsed += elapsed
	s.TotalExecution += execution
	if elapsed > s.MaxElapsed {
		s.MaxElapsed = elapsed
	}
}

// printQueryProfilingSummary logs the top-N slowest query templates sorted by
// total elapsed time. Pass limit <= 0 to show all templates.
func printQueryProfilingSummary(limit int) {
	querySummaryState.Lock()
	if len(querySummaryState.byKey) == 0 {
		querySummaryState.Unlock()
		log.Println("query summary: no query data captured")
		return
	}

	items := make([]querySummary, 0, len(querySummaryState.byKey))
	for _, s := range querySummaryState.byKey {
		items = append(items, *s)
	}
	querySummaryState.Unlock()

	sort.Slice(items, func(i, j int) bool {
		if items[i].TotalElapsed == items[j].TotalElapsed {
			return items[i].MaxElapsed > items[j].MaxElapsed
		}
		return items[i].TotalElapsed > items[j].TotalElapsed
	})

	if limit <= 0 || limit > len(items) {
		limit = len(items)
	}

	log.Printf("query summary: %d distinct query templates (%d shown)", len(items), limit)
	for i := 0; i < limit; i++ {
		s := items[i]
		avgElapsed := time.Duration(0)
		avgExecution := time.Duration(0)
		if s.Count > 0 {
			avgElapsed = s.TotalElapsed / time.Duration(s.Count)
			avgExecution = s.TotalExecution / time.Duration(s.Count)
		}
		log.Printf("query summary #%d [%s] count=%d total_elapsed=%v avg_elapsed=%v max_elapsed=%v avg_execution=%v", i+1, s.Tag, s.Count, s.TotalElapsed, avgElapsed, s.MaxElapsed, avgExecution)
		log.Printf("query summary sql #%d: %s", i+1, s.QueryText)
	}
}

// getDbConnection opens a Couchbase cluster connection using the supplied credentials.
// It waits up to 15 seconds for the bucket to become ready before returning.
func getDbConnection(cred Credentials) (conn CbConnection) {
	log.Println("getDbConnection()")

	conn = CbConnection{}
	connectionString := cred.Cb_host
	bucketName := cred.Cb_bucket
	collection := cred.Cb_collection
	username := cred.Cb_user
	password := cred.Cb_password
	timeout := cred.Cb_timeout_seconds
	if timeout <= 0 {
		timeout = 3600
	}
	options := gocb.ClusterOptions{
		Authenticator: gocb.PasswordAuthenticator{
			Username: username,
			Password: password,
		},
		TimeoutsConfig: gocb.TimeoutsConfig{
			QueryTimeout: time.Duration(timeout) * time.Second,
		},
	}
	if err := configureCapellaTLSOptions(connectionString, &options); err != nil {
		log.Fatal(err)
	}

	cluster, err := gocb.Connect(connectionString, options)
	if err != nil {
		log.Fatal(err)
	}

	conn.Cluster = cluster
	conn.Bucket = conn.Cluster.Bucket(bucketName)
	conn.Collection = conn.Bucket.Collection(collection)
	conn.vxDBTARGET = cred.Cb_bucket + "." + cred.Cb_scope + "." + cred.Cb_collection
	validateQueryParam("vxDBTARGET", conn.vxDBTARGET)

	log.Println("vxDBTARGET:" + conn.vxDBTARGET)

	err = conn.Bucket.WaitUntilReady(15*time.Second, nil)
	if err != nil {
		log.Fatal(err)
	}

	conn.Scope = conn.Bucket.Scope(cred.Cb_scope)
	return conn
}

// configureCapellaTLSOptions loads a CA certificate for Capella (cloud.couchbase.com) clusters.
// It reads CACERT_FILE (path to PEM) and is a no-op when CACERT_REQUIRED is unset or the host is not Capella.
func configureCapellaTLSOptions(connectionString string, options *gocb.ClusterOptions) error {
	// if it isn't a Capella cluster or if CACERT_REQUIRED isn't set, we don't need to configure TLS options
	if !strings.Contains(connectionString, "cloud.couchbase.com") || os.Getenv("CACERT_REQUIRED") == "" {
		return nil
	}
	caPath := os.Getenv("CACERT_FILE")
	if strings.TrimSpace(caPath) == "" {
		return fmt.Errorf("CACERT_FILE must be set for cloud.couchbase.com hosts")
	}

	pemBytes, err := os.ReadFile(caPath)
	if err != nil {
		return fmt.Errorf("failed to read CACERT_FILE %q: %w", caPath, err)
	}

	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(pemBytes) {
		return fmt.Errorf("failed to parse CA cert PEM from %q", caPath)
	}

	options.SecurityConfig = gocb.SecurityConfig{
		TLSRootCAs: roots,
	}
	return nil
}

func queryWithSQLStringSA(scope *gocb.Scope, text string) (rv []string) {
	log.Println("queryWithSQLStringSA(\n" + text + "\n)")
	start := time.Now()

	queryResult, err := scope.Query(
		text,
		newQueryOptions(),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Interfaces for handling streaming return values
	retValues := []string{}

	for queryResult.Next() {
		var row interface{}
		err := queryResult.Row(&row)
		if err != nil {
			log.Fatal(err)
		}
		if s, ok := row.(string); ok {
			retValues = append(retValues, s)
		} else {
			log.Printf("queryWithSQLStringSA: unexpected row type %T, skipping", row)
		}
	}

	finalizeQueryResult("queryWithSQLStringSA", text, start, queryResult)

	return retValues
}

func queryWithSQLStringIA(scope *gocb.Scope, text string) (rv []int) {
	log.Println("queryWithSQLStringIA(\n" + text + "\n)")
	start := time.Now()

	queryResult, err := scope.Query(
		text,
		newQueryOptions(),
	)
	if err != nil {
		log.Fatal(err)
	}

	retValues := make([]int, 0)

	// Stream the values returned from the query into an untyped and unstructred
	// array of interfaces
	for queryResult.Next() {
		var row interface{}
		err := queryResult.Row(&row)
		if err != nil {
			log.Fatal(err)
		}
		switch v := row.(type) {
		case float64:
			retValues = append(retValues, int(v))
		case int:
			retValues = append(retValues, v)
		}
	}

	finalizeQueryResult("queryWithSQLStringIA", text, start, queryResult)

	return retValues
}

func queryWithSQLStringMAP(scope *gocb.Scope, text string) (jsonOut []interface{}) {
	log.Println("queryWithSQLStringMAP(\n" + text + "\n)")
	start := time.Now()

	queryResult, err := scope.Query(
		text,
		newQueryOptions(),
	)
	if err != nil {
		log.Fatal(err)
	}

	rows := make([]interface{}, 0)

	for queryResult.Next() {
		var row interface{}
		err := queryResult.Row(&row)
		if err != nil {
			log.Fatal(err)
		}
		if m, ok := row.(map[string]interface{}); ok {
			rows = append(rows, m)
		} else if s, ok := row.(string); ok {
			rows = append(rows, s)
		}
	}

	finalizeQueryResult("queryWithSQLStringMAP", text, start, queryResult)
	return rows
}
