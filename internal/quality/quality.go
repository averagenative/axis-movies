// Package quality maps a release's source+resolution to a Radarr-style quality
// name and a numeric weight used to rank candidate releases. This is a first
// cut: full quality profiles and custom-format scoring come later.
package quality

import "strings"

func resolutionRank(res string) int {
	switch res {
	case "2160p":
		return 4
	case "1080p":
		return 3
	case "720p":
		return 2
	case "576p", "480p":
		return 1
	default:
		return 0
	}
}

func sourceRank(source string) int {
	switch source {
	case "Remux":
		return 6
	case "BluRay":
		return 5
	case "WEB-DL":
		return 4
	case "WEBRip":
		return 3
	case "HDTV":
		return 2
	case "DVDRip", "DVD":
		return 1
	default: // CAM, TS, unknown
		return 0
	}
}

// Weight ranks a release for sorting; higher is better. Resolution dominates,
// then source.
func Weight(source, resolution string) int {
	return resolutionRank(resolution)*100 + sourceRank(source)*10
}

// Name returns a Radarr-style quality label, e.g. "Bluray-1080p", "Remux-2160p",
// "WEBDL-1080p", "HDTV-720p", "DVD".
func Name(source, resolution string) string {
	prefix := sourcePrefix(source)
	if prefix == "" {
		if resolution != "" {
			return resolution
		}
		return "Unknown"
	}
	if resolution == "" {
		return prefix
	}
	return prefix + "-" + resolution
}

func sourcePrefix(source string) string {
	switch source {
	case "Remux":
		return "Remux"
	case "BluRay":
		return "Bluray"
	case "WEB-DL":
		return "WEBDL"
	case "WEBRip":
		return "WEBRip"
	case "HDTV":
		return "HDTV"
	case "DVDRip", "DVD":
		return "DVD"
	case "CAM":
		return "CAM"
	case "TS":
		return "TELESYNC"
	default:
		return strings.TrimSpace(source)
	}
}
