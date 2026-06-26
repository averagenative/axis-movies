package quality

import "testing"

func TestWeightOrdering(t *testing.T) {
	cases := []struct{ aS, aR, bS, bR string }{
		{"BluRay", "2160p", "WEB-DL", "1080p"}, // higher res wins
		{"Remux", "1080p", "BluRay", "1080p"},  // same res, better source wins
		{"WEB-DL", "1080p", "HDTV", "720p"},
		{"BluRay", "720p", "CAM", "720p"},
	}
	for _, c := range cases {
		if Weight(c.aS, c.aR) <= Weight(c.bS, c.bR) {
			t.Errorf("expected %s/%s > %s/%s", c.aS, c.aR, c.bS, c.bR)
		}
	}
}

func TestName(t *testing.T) {
	cases := map[[2]string]string{
		{"Remux", "2160p"}:  "Remux-2160p",
		{"BluRay", "1080p"}: "Bluray-1080p",
		{"WEB-DL", "1080p"}: "WEBDL-1080p",
		{"HDTV", "720p"}:    "HDTV-720p",
		{"DVD", ""}:         "DVD",
	}
	for in, want := range cases {
		if got := Name(in[0], in[1]); got != want {
			t.Errorf("Name(%q,%q)=%q, want %q", in[0], in[1], got, want)
		}
	}
}
