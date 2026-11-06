# Review: thử lại đổi tên file khi lưu trên Windows (07/11/2026)

Nhánh `fix/thu-lai-doi-ten-windows`. Phạm vi: `server/store.go` (`writeAtomic`, `renameWithRetry`), `rename_retry_windows.go`, `rename_retry_other.go`, `rename_retry_windows_test.go`, `save_failure_regression_test.go`.

## Bối cảnh
- `TestContractWorkSaveFailureDropsCredit` lỗi ngẫu nhiên ~1–2% lượt trên Windows: `os.Rename(file.tmp, file)` bị "Access is denied" khi hai lần lưu liền nhau (trình diệt virus/đánh chỉ mục đang mở file vừa ghi). Log cho thấy cả đường chạy thật (`contracts discard … rename …`) cũng dính: lỗi thật, không chỉ của test.
- Sửa: thử lại rename khi Windows trả ERROR_ACCESS_DENIED (5), ERROR_SHARING_VIOLATION (32), ERROR_LOCK_VIOLATION (33); chờ 1, 2, 4… 64 ms (tối đa ~127 ms, chỉ khi lỗi). Ngoài Windows không thử lại.

## Kết luận reviewer
- Không có lỗi Critical/High. Bắt mã lỗi qua `*os.LinkError` đúng (`errors.As` → `syscall.Errno`).
- Thử lại với Access denied thật (file chỉ đọc, ACL) chỉ trễ ≤127 ms rồi trả nguyên lỗi; logic hoàn tác của nơi gọi vẫn chạy.
- Medium: nhiều kho lưu khi giữ khoá chung (`ContractStore.mu`, `GuildStore.mu`, `Store.mu`) nên lúc thử lại phòng khác có thể chờ theo (Register/nhà ghi 2 file: tối đa ~254 ms). Chấp nhận được vì bình thường chỉ hụt 1–2 lần (1–3 ms).
- Low: test dùng mốc 15 ms, chỉ hỏng nếu goroutine đóng file bị trễ >110 ms.
- Chỉ `writeAtomic` dùng `os.Rename`/`os.WriteFile` ngoài test; `ledger.jsonl`, `admin_audit.jsonl` ghi nối, không đổi tên.

## Đã áp dụng
- Log khi đổi tên được sau thử lại và khi thử hết vẫn hỏng (đo thời gian chờ thực tế).
- Ghi chú khoá chung trong comment `renameWithRetry`; đổi tên hằng số thành `winErr…`, dùng `syscall.ERROR_ACCESS_DENIED`.
- `blockWrite` báo lỗi ngay nếu không dọn được thư mục chặn.

## Kiểm chứng
- `go vet` (Windows và GOOS=linux) sạch.
- Trước sửa: ~4 lỗi/300 lượt. Sau sửa: 500 lượt × 2 test không lỗi; thêm 200 lượt × 3 test (kể cả test mới) không lỗi.

## Câu hỏi còn mở
- Máy chủ chạy thật có loại trừ thư mục `data` khỏi Defender chưa? Loại trừ sẽ bỏ hẳn nguyên nhân gốc.
- Nếu log cho thấy hay phải chờ gần hết 127 ms: cân nhắc rút ngắn ngân sách hoặc chuyển lưu ra khỏi khoá chung.
