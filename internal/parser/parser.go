// Package parser is a clean-room movie release-name parser. It extracts the
// title, year, and quality attributes from scene/p2p release names across the
// common formats (dotted scene, YTS brackets, anime front-group).
//
// It is written from knowledge of release-naming conventions; no regex or data
// is copied from Radarr/Sonarr (GPL).
package parser

import (
	"regexp"
	"strconv"
	"strings"
)

// Release is the parsed result. The first group of fields is high-confidence
// and corpus-tested; Audio/HDR/Edition/Languages are best-effort.
type Release struct {
	Title      string
	Year       int
	Resolution string // 2160p | 1080p | 720p | 576p | 480p | ""
	Source     string // BluRay | Remux | WEB-DL | WEBRip | HDTV | DVDRip | DVD | CAM | TS | ""
	Codec      string // x264 | x265 | XviD | AV1 | VC1 | ""
	Audio      string
	HDR        string
	Edition    string
	Proper     bool
	Repack     bool
	Group      string
	Languages  []string
}

var (
	extRe       = regexp.MustCompile(`(?i)\.(mkv|mp4|avi|m2ts|mov|wmv)$`)
	leadGroupRe = regexp.MustCompile(`^\[([^\]]+)\]\s*`)
	bracketRe   = regexp.MustCompile(`\[([^\]]+)\]`)
	dashGroupRe = regexp.MustCompile(`-([A-Za-z0-9]{2,})$`)
	yearTokRe   = regexp.MustCompile(`^(19\d{2}|20\d{2})$`)

	resRe = regexp.MustCompile(`(?i)\b(2160p|1080p|720p|576p|480p)\b`)
	uhdRe = regexp.MustCompile(`(?i)\b(4k|uhd)\b`)

	remuxRe  = regexp.MustCompile(`(?i)\bremux\b`)
	blurayRe = regexp.MustCompile(`(?i)\b(blu-?ray|bdrip|brrip|bd25|bd50|bdremux)\b`)
	webdlRe  = regexp.MustCompile(`(?i)\bweb[-\s]?dl\b`)
	webripRe = regexp.MustCompile(`(?i)\b(webrip|web)\b`)
	hdtvRe   = regexp.MustCompile(`(?i)\bhdtv\b`)
	dvdripRe = regexp.MustCompile(`(?i)\b(dvdrip|dvd-?rip)\b`)
	dvdRe    = regexp.MustCompile(`(?i)\b(dvd|pal|ntsc)\b`)
	camRe    = regexp.MustCompile(`(?i)\b(hdcam|cam)\b`)
	tsRe     = regexp.MustCompile(`(?i)\b(telesync|ts|hdts)\b`)

	x265Re = regexp.MustCompile(`(?i)\b(x265|h\s?265|h265|hevc)\b`)
	x264Re = regexp.MustCompile(`(?i)\b(x264|h\s?264|h264|avc)\b`)
	xvidRe = regexp.MustCompile(`(?i)\b(xvid|divx)\b`)
	av1Re  = regexp.MustCompile(`(?i)\bav1\b`)
	vc1Re  = regexp.MustCompile(`(?i)\bvc-?1\b`)

	properRe = regexp.MustCompile(`(?i)\bproper\b`)
	repackRe = regexp.MustCompile(`(?i)\brepack\b`)

	// Split hyphen-glued years used by some custom rippers (Title-2008-1080p).
	hyphenYearL = regexp.MustCompile(`(\S)-((?:19|20)\d{2})\b`)
	hyphenYearR = regexp.MustCompile(`\b((?:19|20)\d{2})-(\S)`)

	hdrDVRe   = regexp.MustCompile(`(?i)\b(dolby\s?vision|dovi|\bdv\b)\b`)
	hdr10pRe  = regexp.MustCompile(`(?i)\b(hdr10\+|hdr10plus|hdr10p)\b`)
	hdr10Re   = regexp.MustCompile(`(?i)\b(hdr10|hdr)\b`)
	editionRe = regexp.MustCompile(`(?i)\b(extended|director'?s\.?cut|directors\.?cut|unrated|imax|remastered|final\.?cut|theatrical|special\.?edition|ultimate\.?edition|the\.?criterion\.?collection)\b`)

	// "Hard" quality tokens that reliably appear AFTER the title+year and thus
	// mark the end of the title region. Edition/language words are deliberately
	// excluded: they can appear BEFORE the year ("Extended Edition 2001"), so
	// using them as anchors would hide the year.
	qualityTokenRe = regexp.MustCompile(`(?i)^(` +
		`2160p|1080p|720p|576p|480p|4k|uhd|` +
		`blu-?ray|bdrip|brrip|bd25|bd50|bdremux|remux|web-?dl|webrip|hdtv|dvdrip|dvd|hdcam|cam|telesync|ts|hdts|sdtv|` +
		`x264|x265|h264|h265|hevc|avc|xvid|divx|av1|vc-?1|` +
		`proper|repack|` +
		`hdr10\+|hdr10|hdr|dolby|dovi|dv|10bit|8bit|` +
		`dts|dts-?hd|truehd|atmos|ac3|eac3|ddp?|ddp?5|aac|flac` +
		`)$`)

	// Edition/language tokens stripped from the END of the title region.
	titleStripTok = map[string]bool{
		"extended": true, "unrated": true, "imax": true, "remastered": true,
		"theatrical": true, "edition": true, "limited": true, "uncut": true,
		"german": true, "multi": true, "dual": true, "ita": true, "italian": true,
		"french": true, "truefrench": true, "vostfr": true, "vff": true,
		"korean": true, "japanese": true, "spanish": true, "dl": true,
	}
)

// Parse extracts release attributes from a movie release name.
func Parse(name string) Release {
	r := Release{}
	s := strings.TrimSpace(name)
	s = extRe.ReplaceAllString(s, "")

	r.Group, s = extractGroup(s)

	// Quality/attribute detection runs on a space-normalized copy that keeps
	// dashes (so WEB-DL / DTS-HD survive).
	n := normalize(s)
	r.Proper = properRe.MatchString(n)
	r.Repack = repackRe.MatchString(n)
	r.Resolution = detectResolution(n)
	r.Source = detectSource(n)
	r.Codec = detectCodec(n)
	r.HDR = detectHDR(n)
	r.Audio = detectAudio(n)
	r.Edition = detectEdition(n)
	r.Languages = detectLanguages(n)

	r.Title, r.Year = titleAndYear(n)
	return r
}

func extractGroup(s string) (group, rest string) {
	// Anime front-group: leading [Group].
	if m := leadGroupRe.FindStringSubmatch(s); m != nil {
		return m[1], s[len(m[0]):]
	}
	// YTS/YIFY: trailing bracket tag.
	if br := bracketRe.FindAllStringSubmatch(s, -1); len(br) > 0 {
		last := br[len(br)-1][1]
		up := strings.ToUpper(last)
		if strings.Contains(up, "YTS") || up == "YIFY" {
			return last, strings.Replace(s, "["+last+"]", "", 1)
		}
	}
	// Scene: trailing -GROUP, guarding against false dashes (WEB-DL, DTS-HD).
	if m := dashGroupRe.FindStringSubmatch(s); m != nil {
		if !isNonGroupTail(m[1]) {
			return m[1], s[:len(s)-len(m[0])]
		}
	}
	return "", s
}

// isNonGroupTail rejects trailing dash tokens that are quality fragments, not
// release groups (e.g. the "DL" of WEB-DL, the "HD"/"MA" of DTS-HD MA).
func isNonGroupTail(tok string) bool {
	switch strings.ToUpper(tok) {
	case "DL", "HD", "MA", "EX", "ES", "HD.MA", "HDMA":
		return true
	}
	return false
}

func normalize(s string) string {
	s = strings.NewReplacer(
		".", " ", "_", " ", "[", " ", "]", " ",
		"(", " ", ")", " ", "{", " ", "}", " ",
	).Replace(s)
	s = hyphenYearL.ReplaceAllString(s, "$1 $2")
	s = hyphenYearR.ReplaceAllString(s, "$1 $2")
	return strings.Join(strings.Fields(s), " ")
}

func detectResolution(n string) string {
	if m := resRe.FindStringSubmatch(n); m != nil {
		return strings.ToLower(m[1])
	}
	if uhdRe.MatchString(n) {
		return "2160p"
	}
	return ""
}

func detectSource(n string) string {
	switch {
	case remuxRe.MatchString(n):
		return "Remux"
	case webdlRe.MatchString(n):
		return "WEB-DL"
	case blurayRe.MatchString(n):
		return "BluRay"
	case webripRe.MatchString(n):
		return "WEBRip"
	case hdtvRe.MatchString(n):
		return "HDTV"
	case dvdripRe.MatchString(n):
		return "DVDRip"
	case camRe.MatchString(n):
		return "CAM"
	case tsRe.MatchString(n):
		return "TS"
	case dvdRe.MatchString(n):
		return "DVD"
	}
	return ""
}

func detectCodec(n string) string {
	switch {
	case x265Re.MatchString(n):
		return "x265"
	case x264Re.MatchString(n):
		return "x264"
	case av1Re.MatchString(n):
		return "AV1"
	case xvidRe.MatchString(n):
		return "XviD"
	case vc1Re.MatchString(n):
		return "VC1"
	}
	return ""
}

func detectHDR(n string) string {
	switch {
	case hdr10pRe.MatchString(n):
		return "HDR10+"
	case hdrDVRe.MatchString(n):
		return "DV"
	case hdr10Re.MatchString(n):
		return "HDR10"
	}
	return ""
}

func detectAudio(n string) string {
	for _, a := range []struct{ re, label string }{
		{`(?i)\bdts-?hd(\s?ma)?\b`, "DTS-HD MA"},
		{`(?i)\btruehd\b`, "TrueHD"},
		{`(?i)\b(ddp|dd\+|eac3)\b`, "DDP"},
		{`(?i)\b(ac3|dd)\b`, "AC3"},
		{`(?i)\bdts\b`, "DTS"},
		{`(?i)\bflac\b`, "FLAC"},
		{`(?i)\baac\b`, "AAC"},
	} {
		if regexp.MustCompile(a.re).MatchString(n) {
			label := a.label
			if regexp.MustCompile(`(?i)\batmos\b`).MatchString(n) {
				label += " Atmos"
			}
			return label
		}
	}
	return ""
}

func detectEdition(n string) string {
	if m := editionRe.FindString(n); m != "" {
		return strings.TrimSpace(m)
	}
	return ""
}

func detectLanguages(n string) []string {
	var langs []string
	for _, l := range []struct{ re, name string }{
		{`(?i)\bmulti\b`, "Multi"},
		{`(?i)\b(german|deutsch)\b`, "German"},
		{`(?i)\b(french|vff|vostfr|truefrench)\b`, "French"},
		{`(?i)\b(ita|italian)\b`, "Italian"},
		{`(?i)\b(spanish|castellano)\b`, "Spanish"},
		{`(?i)\bkorean\b`, "Korean"},
		{`(?i)\bjapanese\b`, "Japanese"},
	} {
		if regexp.MustCompile(l.re).MatchString(n) {
			langs = append(langs, l.name)
		}
	}
	return langs
}

// titleAndYear splits the normalized name into the title and release year by
// locating where the quality tokens begin, then taking the last year before it.
func titleAndYear(n string) (string, int) {
	tokens := strings.Fields(n)
	anchor := len(tokens)
	for i, t := range tokens {
		if qualityTokenRe.MatchString(t) {
			anchor = i
			break
		}
	}

	yearIdx, year := -1, 0
	for i := 0; i < anchor; i++ {
		if yearTokRe.MatchString(tokens[i]) {
			yearIdx = i
			year = atoi(tokens[i])
		}
	}

	end := anchor
	if yearIdx >= 0 {
		end = yearIdx
	}
	// Strip trailing edition/language tokens left in the title region.
	for end > 0 && titleStripTok[strings.ToLower(tokens[end-1])] {
		end--
	}
	title := strings.Join(tokens[:end], " ")
	return strings.TrimSpace(title), year
}

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}
