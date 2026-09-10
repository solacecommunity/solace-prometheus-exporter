package version

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

// withBuildInfo temporarily overrides the link-time build metadata and restores it afterwards, so tests can
// assert on concrete label values without depending on how the test binary was built.
func withBuildInfo(t *testing.T, version, commit, buildDate string) {
	t.Helper()
	oldVersion, oldCommit, oldBuildDate := Version, Commit, BuildDate
	Version, Commit, BuildDate = version, commit, buildDate
	t.Cleanup(func() {
		Version, Commit, BuildDate = oldVersion, oldCommit, oldBuildDate
	})
}

// TestCollectorExposesBuildInfo pins the exact metric contract consumers rely on: the fully qualified name, the
// three label names, and a value that is always 1.
func TestCollectorExposesBuildInfo(t *testing.T) {
	withBuildInfo(t, "1.4.2", "abc1234", "2026-08-01")

	want := `
# HELP solace_exporter_build_info Build information of the running exporter. The value is always 1; the build metadata is carried in the labels.
# TYPE solace_exporter_build_info gauge
solace_exporter_build_info{build_date="2026-08-01",commit="abc1234",version="1.4.2"} 1
`

	if err := testutil.CollectAndCompare(NewCollector(), strings.NewReader(want), buildInfoName); err != nil {
		t.Errorf("unexpected build info metric: %v", err)
	}
}

// TestCollectorEmitsExactlyOneSeries guards against the info metric accidentally becoming per-scrape or
// multi-series, which would break `group_left` joins against it.
func TestCollectorEmitsExactlyOneSeries(t *testing.T) {
	if got := testutil.CollectAndCount(NewCollector(), buildInfoName); got != 1 {
		t.Errorf("expected exactly 1 series, got %d", got)
	}
}

// TestFallbacksWhenNotInjected documents the behaviour of a build without -ldflags: the metric is still emitted,
// with values that make it obvious the binary did not come from the release pipeline.
func TestFallbacksWhenNotInjected(t *testing.T) {
	withBuildInfo(t, unknownVersion, unknownCommit, unknownBuildDate)

	want := `
# HELP solace_exporter_build_info Build information of the running exporter. The value is always 1; the build metadata is carried in the labels.
# TYPE solace_exporter_build_info gauge
solace_exporter_build_info{build_date="unknown",commit="unknown",version="dev"} 1
`

	if err := testutil.CollectAndCompare(NewCollector(), strings.NewReader(want), buildInfoName); err != nil {
		t.Errorf("unexpected fallback build info metric: %v", err)
	}
}

// TestCollectorIsRegisterable ensures the collector can join the default-style registry that /metrics serves,
// i.e. that its Describe output is consistent enough for prometheus.Register to accept it.
func TestCollectorIsRegisterable(t *testing.T) {
	registry := prometheus.NewRegistry()
	if err := registry.Register(NewCollector()); err != nil {
		t.Fatalf("collector could not be registered: %v", err)
	}
}
