package parser

import (
	"bufio"
	"os"
	"strings"
	"testing"
)

// TestAuditCollection parses real release names from the file named by
// AXIS_COLLECTION_FILE (one name per line) and reports anomalies. It is an
// auditing tool, not an assertion — it never fails the build, it just logs what
// looks wrong so a real library can be sanity-checked against the parser.
func TestAuditCollection(t *testing.T) {
	path := os.Getenv("AXIS_COLLECTION_FILE")
	if path == "" {
		t.Skip("set AXIS_COLLECTION_FILE to audit a real collection")
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = f.Close() }()

	var total, anomalies int
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		name := strings.TrimSpace(sc.Text())
		if name == "" {
			continue
		}
		total++
		r := Parse(name)
		var why []string
		if strings.TrimSpace(r.Title) == "" {
			why = append(why, "no-title")
		}
		if r.Year == 0 {
			why = append(why, "no-year")
		}
		if r.Resolution == "" && r.Source == "" {
			why = append(why, "no-quality")
		}
		if len(why) > 0 {
			anomalies++
			t.Logf("ANOMALY [%s] %q -> title=%q year=%d res=%q src=%q codec=%q group=%q",
				strings.Join(why, ","), name, r.Title, r.Year, r.Resolution, r.Source, r.Codec, r.Group)
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("scan: %v", err)
	}
	t.Logf("audited %d names, %d anomalies (%.1f%%)", total, anomalies, 100*float64(anomalies)/float64(max(total, 1)))
}
