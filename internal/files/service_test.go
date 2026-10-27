package files

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadFileAndRejectTraversal(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "note.md")
	if err := os.WriteFile(path, []byte("# hello"), 0600); err != nil {
		t.Fatal(err)
	}
	service, err := NewService([]string{root}, 1024, 10)
	if err != nil {
		t.Fatal(err)
	}
	content, err := service.ReadFile(path)
	if err != nil || content != "# hello" {
		t.Fatalf("ReadFile() = %q, %v", content, err)
	}
	if _, err := service.ReadFile(filepath.Join(root, "..", "outside.txt")); err == nil {
		t.Fatal("expected traversal rejection")
	}
}

func TestRejectUnsupportedAndOversizedFiles(t *testing.T) {
	root := t.TempDir()
	pdf := filepath.Join(root, "doc.pdf")
	large := filepath.Join(root, "large.txt")
	if err := os.WriteFile(pdf, []byte("not parsed"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(large, []byte("too large"), 0600); err != nil {
		t.Fatal(err)
	}
	service, err := NewService([]string{root}, 4, 10)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.ReadFile(pdf); err == nil {
		t.Fatal("expected unsupported format error")
	}
	if _, err := service.ReadFile(large); err == nil {
		t.Fatal("expected size limit error")
	}
}

func TestRejectSymlinks(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.txt")
	if err := os.WriteFile(outside, []byte("secret"), 0600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "link.txt")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	service, err := NewService([]string{root}, 1024, 10)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.ReadFile(link); err == nil {
		t.Fatal("expected symlink rejection")
	}
}
