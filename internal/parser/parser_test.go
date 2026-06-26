package parser

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"
)

type corpusEntry struct {
	Release    string `json:"release"`
	Title      string `json:"title"`
	Year       int    `json:"year"`
	Resolution string `json:"resolution"`
	Source     string `json:"source"`
	Codec      string `json:"codec"`
	Proper     bool   `json:"proper"`
	Repack     bool   `json:"repack"`
	Group      string `json:"group"`
}

var nonAlnum = regexp.MustCompile(`[^a-z0-9]+`)

// normTitle strips punctuation/case so the parser is judged on the title region
// it can recover, not on punctuation (colons, apostrophes) absent from the
// release string.
func normTitle(s string) string {
	return nonAlnum.ReplaceAllString(strings.ToLower(s), "")
}

func TestParserCorpus(t *testing.T) {
	data, err := os.ReadFile("testdata/corpus.json")
	if err != nil {
		t.Fatalf("read corpus: %v", err)
	}
	var corpus []corpusEntry
	if err := json.Unmarshal(data, &corpus); err != nil {
		t.Fatalf("parse corpus: %v", err)
	}
	if len(corpus) < 50 {
		t.Fatalf("corpus too small (%d) — expected the full set", len(corpus))
	}

	fails := 0
	for _, c := range corpus {
		got := Parse(c.Release)
		var d []string
		if normTitle(got.Title) != normTitle(c.Title) {
			d = append(d, fmt.Sprintf("title %q != %q", got.Title, c.Title))
		}
		if got.Year != c.Year {
			d = append(d, fmt.Sprintf("year %d != %d", got.Year, c.Year))
		}
		if got.Resolution != c.Resolution {
			d = append(d, fmt.Sprintf("res %q != %q", got.Resolution, c.Resolution))
		}
		if got.Source != c.Source {
			d = append(d, fmt.Sprintf("source %q != %q", got.Source, c.Source))
		}
		if got.Codec != c.Codec {
			d = append(d, fmt.Sprintf("codec %q != %q", got.Codec, c.Codec))
		}
		if got.Proper != c.Proper {
			d = append(d, fmt.Sprintf("proper %v != %v", got.Proper, c.Proper))
		}
		if got.Repack != c.Repack {
			d = append(d, fmt.Sprintf("repack %v != %v", got.Repack, c.Repack))
		}
		if got.Group != c.Group {
			d = append(d, fmt.Sprintf("group %q != %q", got.Group, c.Group))
		}
		if len(d) > 0 {
			fails++
			t.Errorf("FAIL %s\n      %s", c.Release, strings.Join(d, "; "))
		}
	}
	t.Logf("corpus: %d entries, %d passed, %d failed", len(corpus), len(corpus)-fails, fails)
}
