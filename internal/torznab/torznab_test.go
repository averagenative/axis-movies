package torznab

import (
	"strings"
	"testing"
)

const sampleXML = `<?xml version="1.0" encoding="UTF-8"?>
<rss xmlns:torznab="http://torznab.com/schemas/2015/feed">
 <channel>
  <item>
   <title>Blade Runner 2049 2017 2160p BluRay x265-GRP</title>
   <comments>http://indexer/info/1</comments>
   <size>50000000000</size>
   <enclosure url="http://dl/1.torrent" length="50000000000" type="application/x-bittorrent"/>
   <torznab:attr name="seeders" value="100"/>
   <torznab:attr name="leechers" value="5"/>
  </item>
  <item>
   <title>Blade Runner 2049 2017 1080p WEB-DL x264-GRP</title>
   <link>magnet:?xt=urn:btih:abc123</link>
   <torznab:attr name="seeders" value="50"/>
   <torznab:attr name="size" value="8000000000"/>
  </item>
 </channel>
</rss>`

func TestParse(t *testing.T) {
	items, err := Parse(strings.NewReader(sampleXML))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("got %d items, want 2", len(items))
	}
	a := items[0]
	if a.Seeders != 100 || a.Leechers != 5 || a.Size != 50000000000 || a.DownloadURL != "http://dl/1.torrent" {
		t.Fatalf("item0 wrong: %+v", a)
	}
	if a.InfoURL != "http://indexer/info/1" {
		t.Fatalf("item0 infoURL: %q", a.InfoURL)
	}
	b := items[1]
	if b.MagnetURL != "magnet:?xt=urn:btih:abc123" || b.Seeders != 50 || b.Size != 8000000000 {
		t.Fatalf("item1 wrong: %+v", b)
	}
}

func TestSearchURL(t *testing.T) {
	got := SearchURL("http://prowlarr:9696/36/", "/api", "key", "Dune 2021", []int{2000, 2010})
	for _, want := range []string{"http://prowlarr:9696/36/api?", "t=search", "q=Dune+2021", "apikey=key", "cat=2000%2C2010"} {
		if !strings.Contains(got, want) {
			t.Errorf("url %q missing %q", got, want)
		}
	}
}
