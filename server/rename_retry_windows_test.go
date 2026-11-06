//go:build windows

package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Windows: file đích đang bị tiến trình khác mở (như trình diệt virus) thì rename bị từ chối; renameWithRetry
// phải thử lại tới khi file được nhả, nhưng không chờ mãi nếu file bị giữ lâu.
func TestRenameWithRetryWaitsForBriefLock(t *testing.T) {
	dir := t.TempDir()
	dst, src := filepath.Join(dir, "a.json"), filepath.Join(dir, "a.json.tmp")
	for _, f := range []string{dst, src} {
		if err := os.WriteFile(f, []byte("{}"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	h, err := os.Open(dst) // Go mở file không kèm FILE_SHARE_DELETE: chặn đổi tên đè lên nó
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(src, dst); err == nil || !renameRetryable(err) {
		t.Fatalf("file đang mở phải bị từ chối tạm thời, được: %v", err)
	}
	go func() { time.Sleep(15 * time.Millisecond); h.Close() }()
	if err := renameWithRetry(src, dst); err != nil {
		t.Fatalf("nhả file sau 15 ms thì phải đổi tên được: %v", err)
	}

	if err := os.WriteFile(src, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	h2, err := os.Open(dst)
	if err != nil {
		t.Fatal(err)
	}
	defer h2.Close()
	start := time.Now()
	if err := renameWithRetry(src, dst); err == nil {
		t.Fatal("file bị giữ suốt thì phải báo lỗi")
	}
	if d := time.Since(start); d > time.Second {
		t.Fatalf("thử lại quá lâu: %v", d)
	}
}
