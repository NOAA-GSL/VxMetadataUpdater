package main

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"log"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/couchbase/gocb/v2"
)

// ---- helpers ----------------------------------------------------------------

func writeTestCACertPEM(t *testing.T, dir string) string {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate rsa key: %v", err)
	}

	tpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "unit-test-ca"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	der, err := x509.CreateCertificate(rand.Reader, tpl, tpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("failed to create cert: %v", err)
	}

	pemData := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	path := filepath.Join(dir, "ca.pem")
	if err := os.WriteFile(path, pemData, 0o600); err != nil {
		t.Fatalf("failed to write cert file: %v", err)
	}

	return path
}

func resetQuerySummaryState() {
	querySummaryState.Lock()
	defer querySummaryState.Unlock()
	querySummaryState.byKey = map[string]*querySummary{}
}

// captureLog redirects the default logger to a buffer and returns a restore func.
func captureLog(t *testing.T) (*bytes.Buffer, func()) {
	t.Helper()
	var buf bytes.Buffer
	oldWriter := log.Writer()
	oldFlags := log.Flags()
	log.SetOutput(&buf)
	log.SetFlags(0)
	return &buf, func() {
		log.SetOutput(oldWriter)
		log.SetFlags(oldFlags)
	}
}

// ---- setQueryProfilingOptions -----------------------------------------------

func TestSetQueryProfilingOptions(t *testing.T) {
	tests := []struct {
		name           string
		metrics        bool
		mode           string
		slowMs         int
		wantMetrics    bool
		wantMode       gocb.QueryProfileMode
		wantSlowCutoff time.Duration
	}{
		{
			name:           "off_mode_with_metrics",
			metrics:        true,
			mode:           "off",
			slowMs:         250,
			wantMetrics:    true,
			wantMode:       gocb.QueryProfileModeNone,
			wantSlowCutoff: 250 * time.Millisecond,
		},
		{
			name:           "phases_mode_no_metrics",
			metrics:        false,
			mode:           "phases",
			slowMs:         10,
			wantMetrics:    false,
			wantMode:       gocb.QueryProfileModePhases,
			wantSlowCutoff: 10 * time.Millisecond,
		},
		{
			name:           "timings_mode_with_metrics",
			metrics:        true,
			mode:           "timings",
			slowMs:         15,
			wantMetrics:    true,
			wantMode:       gocb.QueryProfileModeTimings,
			wantSlowCutoff: 15 * time.Millisecond,
		},
		{
			name:           "negative_slow_ms_clamped_to_zero",
			metrics:        true,
			mode:           "off",
			slowMs:         -1,
			wantMetrics:    true,
			wantMode:       gocb.QueryProfileModeNone,
			wantSlowCutoff: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			setQueryProfilingOptions(tc.metrics, tc.mode, tc.slowMs)
			if queryProfilingConfig.metricsEnabled != tc.wantMetrics {
				t.Errorf("metricsEnabled: got %v, want %v", queryProfilingConfig.metricsEnabled, tc.wantMetrics)
			}
			if queryProfilingConfig.profileMode != tc.wantMode {
				t.Errorf("profileMode: got %v, want %v", queryProfilingConfig.profileMode, tc.wantMode)
			}
			if queryProfilingConfig.slowQueryCutoff != tc.wantSlowCutoff {
				t.Errorf("slowQueryCutoff: got %v, want %v", queryProfilingConfig.slowQueryCutoff, tc.wantSlowCutoff)
			}
		})
	}
}

// ---- newQueryOptions --------------------------------------------------------

func TestNewQueryOptions_ReflectsCurrentProfilingConfig(t *testing.T) {
	setQueryProfilingOptions(false, "phases", 123)
	opts := newQueryOptions()

	if opts.Adhoc != true {
		t.Fatalf("expected Adhoc=true")
	}
	if opts.Metrics != false {
		t.Fatalf("expected Metrics=false, got %v", opts.Metrics)
	}
	if opts.Profile != gocb.QueryProfileModePhases {
		t.Fatalf("expected Profile=phases, got %v", opts.Profile)
	}
}

// ---- recordQuerySummary -----------------------------------------------------

func TestRecordQuerySummary_AggregatesByTrimmedQueryText(t *testing.T) {
	resetQuerySummaryState()

	recordQuerySummary("q", " SELECT 1 ", 120*time.Millisecond, 80*time.Millisecond)
	recordQuerySummary("q", "SELECT 1", 30*time.Millisecond, 10*time.Millisecond)
	recordQuerySummary("q", "SELECT 2", 50*time.Millisecond, 25*time.Millisecond)

	querySummaryState.Lock()
	defer querySummaryState.Unlock()

	if len(querySummaryState.byKey) != 2 {
		t.Fatalf("expected 2 summary keys, got %d", len(querySummaryState.byKey))
	}

	s := querySummaryState.byKey["q|SELECT 1"]
	if s == nil {
		t.Fatalf("expected summary for key q|SELECT 1")
	}
	if s.Count != 2 {
		t.Fatalf("expected count=2, got %d", s.Count)
	}
	if s.TotalElapsed != 150*time.Millisecond {
		t.Fatalf("expected total elapsed 150ms, got %v", s.TotalElapsed)
	}
	if s.TotalExecution != 90*time.Millisecond {
		t.Fatalf("expected total execution 90ms, got %v", s.TotalExecution)
	}
	if s.MaxElapsed != 120*time.Millisecond {
		t.Fatalf("expected max elapsed 120ms, got %v", s.MaxElapsed)
	}
}

// ---- printQueryProfilingSummary ---------------------------------------------

func TestPrintQueryProfilingSummary(t *testing.T) {
	t.Run("NoData", func(t *testing.T) {
		resetQuerySummaryState()
		buf, restore := captureLog(t)
		defer restore()

		printQueryProfilingSummary(10)

		if !strings.Contains(buf.String(), "query summary: no query data captured") {
			t.Fatalf("expected no-data summary log, got: %s", buf.String())
		}
	})

	t.Run("HonorsLimit", func(t *testing.T) {
		resetQuerySummaryState()
		recordQuerySummary("A", "SELECT 1", 300*time.Millisecond, 200*time.Millisecond)
		recordQuerySummary("B", "SELECT 2", 100*time.Millisecond, 90*time.Millisecond)
		recordQuerySummary("C", "SELECT 3", 50*time.Millisecond, 40*time.Millisecond)

		buf, restore := captureLog(t)
		defer restore()

		printQueryProfilingSummary(2)

		out := buf.String()
		if !strings.Contains(out, "query summary: 3 distinct query templates (2 shown)") {
			t.Fatalf("unexpected summary header: %s", out)
		}
		if strings.Contains(out, "query summary sql #3") {
			t.Fatalf("expected only 2 rows in summary, got output: %s", out)
		}
	})
}

// ---- configureCapellaTLSOptions ---------------------------------------------

func TestConfigureCapellaTLSOptions(t *testing.T) {
	t.Run("NonCloud_NoOp", func(t *testing.T) {
		options := gocb.ClusterOptions{}
		err := configureCapellaTLSOptions("couchbase://localhost", &options)
		if err != nil {
			t.Fatalf("expected no error for non-cloud host, got: %v", err)
		}
		if options.SecurityConfig.TLSRootCAs != nil {
			t.Fatalf("expected TLSRootCAs to remain nil for non-cloud host")
		}
	})

	t.Run("Cloud_RequiresCACertPath", func(t *testing.T) {
		os.Setenv("CACERT_REQUIRED", "true")
		defer os.Unsetenv("CACERT_REQUIRED")

		options := gocb.ClusterOptions{}
		err := configureCapellaTLSOptions("couchbases://foo.cloud.couchbase.com", &options)
		if err == nil {
			t.Fatalf("expected error when CACERT_FILE is missing")
		}
		if !strings.Contains(err.Error(), "CACERT_FILE must be set") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("Cloud_SetsTLSRootCAs", func(t *testing.T) {
		dir := t.TempDir()
		writeTestCACertPEM(t, dir)
		os.Setenv("CACERT_FILE", dir+"/ca.pem")
		os.Setenv("CACERT_REQUIRED", "true")
		defer os.Unsetenv("CACERT_FILE")
		defer os.Unsetenv("CACERT_REQUIRED")

		options := gocb.ClusterOptions{}
		err := configureCapellaTLSOptions("couchbases://foo.cloud.couchbase.com", &options)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if options.SecurityConfig.TLSRootCAs == nil {
			t.Fatalf("expected TLSRootCAs to be configured")
		}
	})
}
