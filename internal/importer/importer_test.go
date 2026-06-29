package importer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNaming(t *testing.T) {
	if got := FolderName("The Matrix", 1999); got != "The Matrix (1999)" {
		t.Errorf("folder: %q", got)
	}
	if got := FileName("The Matrix", 1999, "Bluray-1080p", ".mkv"); got != "The Matrix (1999) [Bluray-1080p].mkv" {
		t.Errorf("file: %q", got)
	}
	// illegal characters sanitized
	if got := FolderName("Mission: Impossible", 1996); got != "Mission - Impossible (1996)" {
		t.Errorf("sanitize: %q", got)
	}
}

func TestScanPicksLargestSkipsSample(t *testing.T) {
	dir := t.TempDir()
	write := func(name string, size int) {
		if err := os.WriteFile(filepath.Join(dir, name), make([]byte, size), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("movie.mkv", 5000)
	write("sample.mkv", 100)
	write("readme.nfo", 10)
	write("movie-sample.mp4", 50)

	files, err := ScanVideoFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Fatalf("got %d video files, want 1 (largest, no samples): %+v", len(files), files)
	}
	if filepath.Base(files[0].Path) != "movie.mkv" {
		t.Fatalf("picked %q, want movie.mkv", files[0].Path)
	}
}

func TestImportHardlinks(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "download", "movie.mkv")
	if err := os.MkdirAll(filepath.Dir(src), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(src, []byte("video-bytes"), 0o644); err != nil {
		t.Fatal(err)
	}
	lib := filepath.Join(root, "library")
	dest := DestPath(lib, "The Matrix", 1999, "Bluray-1080p", ".mkv")

	if err := Import(src, dest); err != nil {
		t.Fatalf("import: %v", err)
	}
	// dest exists with same content
	b, err := os.ReadFile(dest)
	if err != nil || string(b) != "video-bytes" {
		t.Fatalf("dest content wrong: %q err=%v", b, err)
	}
	// it is a hardlink (same inode), not a copy
	si, _ := os.Stat(src)
	di, _ := os.Stat(dest)
	if !os.SameFile(si, di) {
		t.Fatal("expected hardlink (same inode)")
	}
	// idempotent: re-import succeeds
	if err := Import(src, dest); err != nil {
		t.Fatalf("re-import: %v", err)
	}
}
