//go:build !windows

package main

// Ngoài Windows, rename đè file là nguyên tử và không bị tiến trình khác giữ file chặn: không cần thử lại.
func renameRetryable(error) bool { return false }
