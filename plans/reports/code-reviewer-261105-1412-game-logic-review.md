# Rà soát logic game: server Go (05/11/2026)

Phạm vi: toàn bộ `server/*.go` (~13k dòng không tính test) + `client/js/tug.js`, đối chiếu PLAN_LOGIC_GAME.md, PLAN.md, PLAN_THE_GIOI_3D.md. 4 agent rà song song theo mảng, lead tự đọc lại code để xác minh các lỗi nặng.

Hạn chế: máy không cài Go → **không build, không chạy test**. Mọi kết luận từ đọc code. Cột "XM": ✅ = lead đã tự đọc lại code xác nhận; 🔎 = agent truy vết đầy đủ, lead chưa đọc lại; ⏱ = phụ thuộc thời điểm/lỗi ghi đĩa.

## Cập nhật 05/11/2026 chiều: đã sửa C1–C4 (chưa commit)

Kiểm thử sau khi cài Go 1.27.3 (winget):
- `go build` và `go vet` sạch.
- Lần chạy đầu: 46 PASS, 31 FAIL **chỉ** do Windows không xoá được TempDir (`ledger.jsonl`/`admin_audit.jsonl` mở mà không đóng; code gốc cũng bị vậy).
- Đã thêm `Store.Close()` và `Moderation.Close()`, gọi khi test dọn dẹp. Chạy lại `go test -count=1 -timeout 25m ./...` (~11 phút): **77/77 PASS**.
- Chưa chạy `-race`: cần cgo + gcc, máy chưa có.

- **C1:** thêm `validAccountID` (24 ký tự hex thường) trong `store.go`. Chặn tại `Store.Account`, `SocialStore.getLocked`/`saveLocked`, `HomeStore.Get`/`saveLocked`, `Hub.resolve`, guild `target()`. Chặn sớm trong `Hub.social` cho các lệnh dùng thẳng `m.ID`.
- **C2:** lệnh `sell` trả lỗi sớm. Thêm `Room.safely` (recover cho inbox và tick); `DoWait` dùng `defer close(done)`; `Session.handleSafe` recover trong `readPump` rồi ngắt phiên.
- **C3:** `Room.remove` gọi `abandonDig(p)` (hàm này cũng huỷ lượt câu cá).
- **C4:** `ContractStore.Work(id, worker, homeOwner, now)` kiểm chủ nhà trước khi cộng tiến độ.
- **Test hồi quy:** `server/security_regression_test.go` (C1, C2, C3); phần C4 nằm trong `TestContractStoreRules` (`v11_test.go`).

## Nghiêm trọng (sửa trước khi cho người ngoài chơi)

| # | Lỗi | Vị trí | Kịch bản | XM |
|---|---|---|---|---|
| C1 | **Path traversal ở mạng xã hội** | `social_hub.go:121` (`friend_remove` dùng `m.ID` thô), `social_hub.go:69` (`resolve` nhận mọi chuỗi 24 ký tự), `social.go:117` | Gửi `friend_remove id="../accounts/<idNạnNhân>"`: server đọc file tài khoản thành `Social` rồi ghi đè lại. Sau restart nạn nhân mất ví, đồ, cấp. Có thể nhắm cả `users.json` (hỏng đăng nhập toàn server) | ✅ |
| C2 | **Bán giun ID lạ làm sập cả server** | `actions.go:256` | `act sell id="xxx"` ở sạp giun: `wormByID(m.ID).Name` deref nil trước khi trả `err`. Không có `recover()` ở đâu cả, panic trong goroutine phòng làm chết process | ✅ |
| C3 | **Re-roll giun/cá vô hạn khi đổi chuồng** | `room.go:322` `remove()` không gọi `abandonDig`; `room.go:317` `canLeave` không chặn khi đang đào | `dig_start` (reply lộ loài giun) → `switch_room` (giữ nguyên vị trí) → `dig_start` lại ngay, không hồi đất. Lặp tới Giun cầu vồng. Tương tự với câu cá. Trái plan §4.3. Sửa: gọi `r.abandonDig(p)` đầu `remove()` | ✅ |
| C4 | **Thợ được trả công mà không xây chuồng cho chủ** | `contracts.go:378` | Thợ đứng ở nền chuồng trong nhà bất kỳ, gửi `contract_work`. `Work()` cộng tiến độ và lưu **trước khi** kiểm `k.Owner != h.Owner`. Đủ `Need` lần → "done", trả công, chủ mất ký quỹ và vật liệu, không được hoàn | ✅ |
| C5 | **Một tài khoản có 2 phiên điều khiển** | `hub.go:260-288` | `Hello` đọc `old := h.sessions[id]` rồi nhả khoá, ghi lại sau. Hai `hello` cùng lúc → 2 Player dùng chung `*Account` ở 2 phòng → nhân đồ, hoặc `fatal error: concurrent map writes` (sập server). Trái PLAN.md §6.3 | ✅ ⏱ |

## Cao

| # | Lỗi | Vị trí | Ghi chú | XM |
|---|---|---|---|---|
| H1 | `switch_room "new"` tạo phòng vô hạn, phòng public không bao giờ đóng | `hub.go:429` | Mỗi phòng một goroutine 20 Hz kèm xe buýt và NPC. Một client lặp lệnh là ngốn hết CPU/RAM (DoS) | ✅ |
| H2 | Câu cá không có giới hạn ngày | `fishing.go:136-144`, `fishing.go:92` | ~2.800 thóc/giờ, không trần. Vượt ngân sách ~1.150 thóc/ngày (§17). Plan chưa có đặc tả câu cá | ✅ |
| H3 | Chuyển phòng/vùng bằng goroutine không tuần tự | `actions.go:92,109`, `v11_room.go:113,501`, `hub.go:449-486` | Bấm đúp "về nhà" hoặc đổi chuồng đúng lúc xe liên trại chạy → player bị `add` 2 lần: ghost, đếm sai, hoặc có mặt ở 2 phòng (giống C5) | 🔎 ⏱ |
| H4 | Bộ đếm quà shipper (+1 thóc, tối đa 5/ngày) reset khi đổi kênh | `npc.go:1010-1019`, `room.go:205` | `npcMemo` tạo mới mỗi lần `add` → farm thóc không giới hạn. Các giới hạn trêu/cameo/xe ôm/hụt hơi cũng reset | 🔎 |
| H5 | Chỉ message `in` có rate limit | `hub.go:206-240` | Mỗi `act` thành công ghi toàn bộ account xuống đĩa ngay trên goroutine phòng. Spam `prefs`/`title`/`report` làm tick phòng chậm cho cả 50 người; spam report đẩy báo cáo thật ra khỏi hàng đợi (`social.go:345`) | 🔎 |

## Trung bình

| # | Lỗi | Vị trí | XM |
|---|---|---|---|
| M1 | Lên cấp do ăn giun không gọi `r.levelUp`, nên không nhận quà cấp ngay (trái §12) | `actions.go:259-272` | ✅ |
| M2 | Cất luống rau nhà (pickup) hoặc đổi lô đất làm cây mồ côi: mất hạt, không thu hoạch được | `home.go:253`, `estate.go:146`, `logic_actions.go:204` | ✅ |
| M3 | Kho đầy khi lên cấp thì toàn bộ vật phẩm quà cấp bị bỏ vĩnh viễn, không báo (trái PLAN.md §7.1) | `items.go:224-241` | 🔎 |
| M4 | Lưu store phụ trước account: nếu lưu account lỗi thì được miễn phí (`contract_create` → ký quỹ hoàn 2 lần; `guild_create` → hội free; `land_buy` → đất free) | `contracts.go:331`, `guild.go:681`, `logic_actions.go:146` | 🔎 ⏱ |
| M5 | `ContractStore.Work` lưu lỗi nhưng công vẫn nằm trong hàng chờ → trả 2 lần hoặc vừa trả vừa hoàn; `Homes.Save` lỗi thì mất chuồng nhưng thợ vẫn được trả | `contracts.go:211-221`, `contracts.go:396` | 🔎 ⏱ |
| M6 | `GuildStore.Do` không hoàn tác thay đổi trong bộ nhớ khi lưu lỗi → quỹ hội tăng mà không trừ tiền | `guild.go:201-212`, `guild.go:698` | 🔎 ⏱ |
| M7 | Lời mời chuyển chủ hội không hết hạn, không bị xoá khi người được mời rời hội hoặc bị kick | `guild.go:458-499` | 🔎 |
| M8 | `LastSeen` của chủ hội chỉ cập nhật khi mở bảng hội → chủ chơi hằng ngày vẫn bị phó hội `claim` sau 30 ngày | `guild.go:270-276` | 🔎 |
| M9 | Không áp giới hạn người theo khu (sảnh 50, khu khác 25); `Zone.Capacity` không ai đọc | `world.go:190`, `config.go:270` | 🔎 |
| M10 | Mất mạng trong lúc đếm ngược PK thì trận không huỷ; `pk_end` gửi vào phiên chết nên nối lại không thấy kết quả | `room.go:278`, `pk.go:251`, `pk.go:327` | 🔎 |
| M11 | Danh hiệu tuần bị mất nếu người thắng không đăng nhập trong tuần kế tiếp | `weekly.go:68-117` | 🔎 |
| M12 | Điểm danh lỗi im lặng khi kho đầy (ngày 2/4/5 có quà vật phẩm), sang hôm sau thì mất chuỗi | `daily.go:47`, `npc_director.go:44` | 🔎 |
| M13 | `withHome` hết hạn 3 giây nhưng closure vẫn chạy sau đó → data race trên `p.acc`, đất lên hạng mà không bị trừ tiền; `zones["home"]` không dựng lại | `estate.go:230-261` | 🔎 ⏱ |
| M14 | `finish` xoá phiên đào/câu trước `txn`; nếu txn lỗi thì mất con giun/cá và không tính hồi | `actions.go:190`, `fishing.go:183` | 🔎 |

## Thấp (tóm tắt)

- Giao dịch bỏ qua giới hạn túi 60 giun (`trade.go:177`); không idempotent (`trade.go:135`); không đóng khi đi bộ sang khu khác (`actions.go:543`).
- Hệ số kích thước giun dùng vị trí trong danh sách thay vì bậc hiếm: rong ×1.45 thay vì ×1.6, bí ngô ×1.0 (`account.go:487`).
- Read limit 4096 byte có thể nhỏ hơn 400 sự kiện giằng co hợp lệ (`hub.go:181` vs `fishing.go:76`).
- Lô trưng bày lộ account ID đầy đủ và hiện cả nhà riêng tư hoặc của người đã chặn mình (`showcase.go:68`). Nên sửa cùng C1.
- Nhà xây xong khi chủ offline không được lưu hoặc thông báo lúc mở phòng (`room.go:263`).
- Lời mời vào hội không hết hạn, không kiểm tra chặn (`guild.go:224,405`).
- Chống auto-click chỉ là khoảng cách tối thiểu 60 ms; 2 tài khoản thông đồng có thể farm bảng PK tuần (`pk.go:158`, `pk.go:314`).
- Hành khách có thể bị đặt vào trong tường khi xuống xe (`v11_room.go:158,187`, `npc.go:555`).
- `DoWait` trên phòng nhà đã đóng có thể treo vĩnh viễn (`room.go:171`, `hub.go:271`).
- `xeom_call` gọi lặp lại thì một người giữ hết xe ôm (`npc_director.go:699`).
- Cứu đói bỏ qua `Items`: ôm cá vàng vẫn nhận cám, việc "dọn ổ rơm" farm được 5 thóc/30 giây (`account.go:675`).
- `restore` xoá ledger chưa ghi (`actions.go:502`); không fsync (`store.go:85`); rid rỗng được chấp nhận (`account.go:404`).
- Bạn ngồi ổ đã mất kết nối vẫn được tính thưởng nghỉ cạnh bạn (`npc_director.go:297`).
- Client làm tròn thời điểm nhấn giằng co có thể lệch 1 bước so với server, hiếm (`dig.js:89`, `fish.js:60`).

## Đã kiểm và đúng plan (rút gọn)

- `tug.js` khớp 1-1 với `dig.go`; RNG mulberry32 khớp (agent chạy Node: 2.000 seed × 50 lần rút, 0 lệch).
- Bảng xác suất giun và trứng đều có tổng bằng 1. May mắn có trần 0,10. Giun phát sáng ×1,5 và mưa −30% đúng.
- Bộ `live` khớp §17: độ no, chạy nhanh, quà khởi nghiệp, ăn sống, 120 lượt đào/ngày. Chuyển đổi dữ liệu cũ (§19) đúng.
- Trade: khoá/xác nhận đúng, sửa đề nghị thì mở khoá cả hai bên, đủ cấp 3, chặn tự giao dịch và người đã chặn.
- Hợp đồng ký quỹ: hoàn tiền đúng 1 lần, thợ từ cấp 7, uy tín giới hạn mỗi cặp mỗi tuần.
- Hội: cấp 10 + 1.000 thóc, tối đa 30 người, quỹ không rút được.
- Xe buýt/xe ôm: giá, lượt miễn phí, phụ thu mưa, hoàn vé 1 lần. Giới hạn lời NPC trong một phiên đúng.
- Chặn người dùng được tôn trọng ở chat, DM, PK, mời ghế, thăm nhà. Phân quyền admin đúng.

## Thứ tự sửa đề xuất

1. C1, C2, C3, C4: sửa nhỏ, cục bộ (validate ID 24 hex; return sớm khi lỗi; `abandonDig` trong `remove`; kiểm chủ trước khi `Work`). Thêm `recover()` trong vòng lặp phòng và trong `handle`.
2. C5, H3: tuần tự hoá việc gắn phiên và chuyển phòng theo từng tài khoản.
3. H1, H5: giới hạn tạo phòng và rate limit cho `act`/social.
4. H2, H4: chốt luật câu cá; lưu bộ đếm shipper vào Account.
5. Nhóm M4–M6 (lưu store phụ trước account): đổi thứ tự lưu và hoàn tác cho nhất quán.
6. Mỗi lỗi kèm một test hồi quy; chạy `go test -race -timeout 25m ./...` sau khi cài Go.

## Câu hỏi chưa giải quyết

1. Câu cá có giới hạn ngày không: chung với 120 lượt đào hay riêng? Ngân sách thóc/XP bao nhiêu?
2. Quà cấp khi kho đầy: đưa vào hàng chờ (theo PLAN.md §7.1) hay bỏ như code hiện tại?
3. Kho 200 có tính nông sản và cá không? Plan §11 chỉ nói vật liệu và hạt.
4. Account ID có cần giữ bí mật không? Ảnh hưởng mức độ của lỗi lộ ID ở lô trưng bày.
5. `home_upgrade` từng bậc (tổng 4.200 thóc) vẫn chạy song song với `BuyLot` (3.000 thóc): có bỏ không?
6. Khách nghỉ cạnh ổ trong nhà người khác có được hồi độ no không, hay chỉ chủ nhà?
7. Bỏ trận PK (đi ra khỏi sân) có tính vào bảng xếp hạng PK tuần không?
8. Xe buýt: plan §7.1 nói 2 xe (3 xe khi > 30 người ngoài trời), bán kính lên xe 3 m; code cố định 3 xe, 5 m. Có phải cố ý?
