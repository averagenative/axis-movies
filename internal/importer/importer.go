// Package importer moves completed downloads into the movie library: it scans a
// download folder for the feature video file, computes the destination path from
// a naming scheme, and hardlinks (or copies across filesystems) it into place.
package importer

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
)

var videoExts = map[string]bool{
	".mkv": true, ".mp4": true, ".avi": true, ".m2ts": true, ".ts": true,
	".mov": true, ".wmv": true, ".mpg": true, ".mpeg": true, ".flv": true, ".webm": true,
}

// VideoFile is a candidate file found in a download folder.
type VideoFile struct {
	Path string
	Size int64
}

// ScanVideoFiles returns the video files under root (recursively), largest
// first, skipping obvious "sample" files. A plain file path is also accepted.
func ScanVideoFiles(root string) ([]VideoFile, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	var out []VideoFile
	if !info.IsDir() {
		if isVideo(root) {
			out = append(out, VideoFile{Path: root, Size: info.Size()})
		}
		return out, nil
	}
	err = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !isVideo(path) {
			return err
		}
		if strings.Contains(strings.ToLower(filepath.Base(path)), "sample") {
			return nil
		}
		fi, err := d.Info()
		if err != nil {
			return err
		}
		out = append(out, VideoFile{Path: path, Size: fi.Size()})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Size > out[j].Size })
	return out, nil
}

func isVideo(path string) bool {
	return videoExts[strings.ToLower(filepath.Ext(path))]
}

// FolderName builds the movie folder name, e.g. "The Matrix (1999)".
func FolderName(title string, year int) string {
	name := Sanitize(title)
	if year > 0 {
		name = fmt.Sprintf("%s (%d)", name, year)
	}
	return name
}

// FileName builds the destination filename, e.g.
// "The Matrix (1999) [Bluray-1080p].mkv".
func FileName(title string, year int, qualityName, ext string) string {
	base := FolderName(title, year)
	if qualityName != "" && qualityName != "Unknown" {
		base += " [" + qualityName + "]"
	}
	return base + ext
}

// DestPath is the absolute destination for an imported file.
func DestPath(rootFolder, title string, year int, qualityName, ext string) string {
	return filepath.Join(rootFolder, FolderName(title, year), FileName(title, year, qualityName, ext))
}

// Sanitize removes characters that are illegal or awkward in file paths.
func Sanitize(s string) string {
	r := strings.NewReplacer(
		"/", "", "\\", "", ":", " -", "*", "", "?", "", "\"", "",
		"<", "", ">", "", "|", "",
	)
	return strings.TrimSpace(r.Replace(s))
}

// Import places src at dest, creating parent dirs. It hardlinks when possible
// and falls back to a copy across filesystem boundaries. An existing dest is
// replaced (idempotent re-import).
func Import(src, dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	if _, err := os.Stat(dest); err == nil {
		if err := os.Remove(dest); err != nil {
			return err
		}
	}
	if err := os.Link(src, dest); err != nil {
		if errors.Is(err, syscall.EXDEV) {
			return copyFile(src, dest)
		}
		return fmt.Errorf("hardlink: %w", err)
	}
	return nil
}

func copyFile(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	tmp := dest + ".axis-partial"
	out, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, dest)
}
