//go:build windows

package main

import (
	"errors"
	"syscall"
)

// Windows: đổi tên đè lên file vừa ghi có thể bị từ chối tạm thời khi trình diệt virus hoặc dịch vụ đánh chỉ mục
// đang mở file đó (không cho xoá/đổi tên). Các lỗi này hết sau vài mili giây nên đáng thử lại.
const (
	winErrSharingViolation syscall.Errno = 32 // gói syscall không đặt tên cho hai mã này
	winErrLockViolation    syscall.Errno = 33
)

func renameRetryable(err error) bool {
	var en syscall.Errno
	if !errors.As(err, &en) {
		return false
	}
	return en == syscall.ERROR_ACCESS_DENIED || en == winErrSharingViolation || en == winErrLockViolation
}
