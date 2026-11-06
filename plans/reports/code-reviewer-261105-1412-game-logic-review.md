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

## Cập nhật 05/11/2026 tối: đã sửa C5, H1–H5 (chưa commit)

- **C5:** `Hub.Hello` giữ khoá theo tài khoản (`accountLock`) suốt lệnh; `kick` của admin dùng cùng khoá.
- **H3:** `Session.moveMu` tuần tự hoá mọi lần đổi phòng: `Switch`, `transferOne` (từ `TransferRegion`), `Visit`, `VisitGuild`, `LeaveHome`, `Leave`, việc thay phiên trong `Hello` và `kick`. Phiên đã kick/đóng thì bỏ qua và dọn `lastPublic`. `Room.add` đánh dấu `goneAt` nếu socket đã đóng.
- **H1:** `switch_room "new"` dùng lại chuồng trống của vùng (`emptyOrNewRoomLocked`). Số phòng ≤ số phòng có người + 1.
- **H2:** trần câu cá `Balance.FishDailyLimit`, mặc định `FishDailyDefault` = 60 con/ngày (~720 thóc). **Số tạm chọn, cần chủ dự án chốt.**
- **H4:** bộ đếm quà shipper chuyển sang `Account.ShipTipDay/ShipTips` (lưu bền, không reset khi đổi kênh).
- **H5:** xô rò 20 msg/s, chứa 40 cho mọi message ngoài `in`/`ping`. Vượt 40: `act` nhận `res` lỗi, message khác bị bỏ. Vượt 80: bỏ hẳn. Báo cáo: tối đa 10 báo cáo đang mở/người/24 giờ; cắt hàng đợi bỏ báo cáo đã xử lý trước.
- **Test hồi quy:** `server/concurrency_limits_regression_test.go` (7 test). Chạy 3 test C5/H1/H3 trên code gốc (worktree tạm): cả 3 fail đúng lỗi (2 nhân vật/1 tài khoản; 7 phòng; sĩ số 2 với 1 người). Trên code mới đều PASS.
- **Review độc lập:** không có deadlock. Thứ tự khoá: khoá tài khoản → `moveMu` → `DoWait` phòng; goroutine phòng không bao giờ lấy hai khoá này. Cần giữ bất biến này: không được `DoWait` vào phòng đã dừng khi đang giữ `moveMu`.
- Toàn bộ test sau khi sửa theo review: `go test -count=1 -timeout 25m ./...` **84/84 PASS** (~11 phút, không `-race`).
- Chưa làm trong nhóm này: các bộ đếm NPC khác (trêu, cameo, xe ôm, hụt hơi) vẫn đặt lại khi đổi kênh. Chỉ ảnh hưởng tần suất NPC, không tạo ra thóc.

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

---

# Lượt rà 2 (05/11/2026 tối)

3 góc mới: (A) đối chiếu ~170 luật/con số plan ↔ code; (B) client JS ↔ giao thức server; (D) soi sâu PK, xe/vé, trứng/pet, trồng cây/đất, mốc ngày. Không lặp lại các mục ở trên. XM: ✅ = lead tự đọc lại code; 🧪 = agent chứng minh bằng test tạm (worktree đã xoá); 🔎 = agent truy vết.

## Chặn hẳn tính năng / hồi quy (sửa trước)

| # | Lỗi | Vị trí | XM |
|---|---|---|---|
| R1 | **"Bán hết" vượt giới hạn tần suất (hồi quy do H5):** client bán từng con, tối đa 60 act liên tiếp → quá 40 thì dừng với "Thao tác nhanh quá" | `client/js/panels.js:188-200`, `server/hub.go` rateMsg | ✅ |
| B1 | **Giao dịch luôn bị huỷ:** mỗi lần nhận `trade`, `onTrade` gọi lại `openSheet` → chạy callback đóng cũ → gửi `trade_cancel` | `client/js/v11.js:235-253`, `client/js/ui.js:23` | ✅ |
| B2 | **Chọn điểm xe ôm làm huỷ chuyến:** bấm chọn khu → `render()` → `openSheet` → callback đóng gửi `xeom_decline` | `client/js/map.js:117-125` | ✅ |
| B3 | **Hành khách không xuống được xe người khác:** server có `seat_leave`/`seat_kick`, client không có nút | `server/v11_room.go:451,458` | ✅ |
| D1 | **PK Úp ngửa treo vô hạn:** người được chọn không bấm thì trận không bao giờ kết thúc; đối thủ chỉ thoát bằng bỏ trận (bị xử thua, bên kia +thắng, +bảng tuần, +điểm hội); sân PK bị coi là bận nên huỷ trận biểu diễn | `server/pk.go:273-282` | ✅🧪 |

### Cập nhật: đã sửa R1, B1–B3, D1 (chưa commit)

- **R1:** `sell` nhận số lượng `i` (`Account.SellWorms`, một giao dịch, một dòng sổ cái, nhiệm vụ ngày vẫn đếm từng con); "Bán hết" gửi một lệnh. Bán lẻ không gửi `i` → 1 con như cũ.
- **B1, B2:** `openSheet` bỏ qua callback đóng cũ khi vẽ lại đúng bảng đang mở (giao dịch, bản đồ, đổi tab). Kèm sửa phụ: mở DM người khác khi DM đang mở không còn xoá `G.dmWith`; `pickDestination` huỷ pick cũ trước khi gán pick mới.
- **B3:** thẻ "Đi nhờ xe · Xuống xe" (`seat_leave`) khi ngồi xe người chơi; danh sách "Quanh đây" của tài xế có "Mời xuống xe" (`seat_kick`).
- **D1:** `Match.PickBy`: người chọn có `CoinPickTimeout` = 10 giây, quá hạn máy chủ chọn ngẫu nhiên (chốt mặc định, câu hỏi 5 bên dưới). Lượt Gà Máy dùng cùng cơ chế (`CoinBotPickDelay` 900 ms), bỏ goroutine `time.Sleep`. `pk_round`/`pk_resume` gửi `pickLeft`; client hiện đếm ngược. Hành vi mới: người chọn mất mạng thì máy chọn thay sau 10 giây mỗi lượt.
- Test mới (`server/feature_blockers_regression_test.go`): `TestSellAllInOneMessage`, `TestSellWormsCounts`, `TestCoinPickTimeoutAutoPicks`, `TestCoinBotPicksAndLatePickRejected`. B1–B3 là lỗi client, chưa kiểm trên trình duyệt (chỉ `node --check`).
- Code review: không lỗi nghiêm trọng; đã áp dụng góp ý (đồng hồ đếm giữ tham chiếu phần tử khi nối lại, chỉ vẽ lại bảng bạn bè khi còn mở, huỷ pick cũ, thêm test Gà Máy và chọn trễ). Còn nhỏ: danh sách "Quanh đây" không tự cập nhật khi có người lên/xuống xe (bấm sẽ nhận lỗi rõ ràng).

## Kinh tế / farm

| # | Lỗi | Vị trí | XM |
|---|---|---|---|
| D2 | Luống rau nhà không giới hạn: Tổ nhỏ đặt được 39 luống → ~2.500 thóc/giờ. **Cần chốt:** giới hạn luống theo hạng đất, trần bán, hay giảm giá | `home.go:219`, `logic_actions.go:204`, `items.go:140` | 🧪 |
| D3 | Thắng Gà Máy vẫn cộng điểm hội tuần và danh hiệu "Cao thủ PK" | `server/pk.go:344` | ✅🧪 |
| A7 | Thắng PK Úp ngửa (may rủi) cũng tính bảng tuần/điểm hội (PLAN.md:399 nói không) | `server/pk.go:341-344` | 🔎 |
| — | Thân thiết thú cưng farm được (tưới giúp +1/luống không trần; ngồi ổ cạnh bạn +60/giờ) | `logic_actions.go:122` | 🔎 |

## Lệch plan (trung bình)

| # | Lỗi | Vị trí | XM |
|---|---|---|---|
| A1 | Xe liên trại chạy 20 giây sau người đặt vé đầu, không theo lịch 3 phút (README cũng ghi sai) | `v11_room.go:353` | ✅ |
| A2 | Nhà luôn giới hạn 8 khách; plan: rơm 8 / gỗ 12 / gạch 16, panel hiện "tối đa 16" | `social_hub.go:344` | ✅ |
| A3 | Giao dịch không trao đổi được vật phẩm (plan §11 cho phép vật liệu, hạt, nông sản từ cấp 3) | `trade.go:15` | ✅ |
| A4 | Gà con nở trước v2 có `Species` rỗng → không bao giờ có kỹ năng "Mỏ Thính" (client vẫn hiện +4%) | `pets.go:55`, thiếu migration | ✅ |
| A5 | Cấp tối đa 20 (plan ghi 10); cấp 11–20, lô đồi, nhà 2 tầng chưa vào ngân sách §17 — có vẻ cố ý, cần cập nhật plan | `config.go:274` | 🔎 |

## Client (trung bình)

- B4: đổi kênh/mất mạng giữa lượt đào/câu thì màn giằng co vẫn chạy, gà đứng im; `G.cancelDig` không bao giờ được gọi, câu cá không có huỷ (`dig.js:122`, `fish.js`, `main.js:253`).
- B5: sau khi nối lại giữa trận PK, listener phím còn sót → gõ "e"/dấu cách trong chat bị nuốt (`panels.js:631-684`).
- B7: giá bán hiển thị khác giá server trả (giun hot +10%, kỹ năng Tay Buôn) (`panels.js:172`).
- B8: joystick gửi tới 60 `in`/giây, server bỏ quá 30 → lệnh dừng có thể mất, gà trôi thêm ~2 m (`main.js:899`).
- B9: bảng xe liên trại vẫn ghi "3 lượt/ngày, reset 0:00" (thực tế 2 lượt, hồi 5 phút) (`v11.js:136`).
- B10: hai panel cùng tên modal `bus` → panel bến làng bị thay bởi panel xe liên trại (`npc.js:276`, `v11.js:122`).
- B11–B13: đếm ngược nhà/tiệc cũ dần; hẹn giờ vườn dùng đồng hồ máy; toast "Đã kết nối lại" mỗi lần đổi phòng.

## Thấp (tóm tắt)

- PK: mời lại mỗi 10 giây giữ người kia "bận" mãi (`pk.go:113,117`); `pk_bot` không huỷ lời mời đang chờ (`pk.go:79`).
- Xe: `drive_region` dùng danh sách khách lúc bấm, khách vừa xuống vẫn bị chở đi (`v11_room.go:466-501`); lên xe buýt/xe ôm trước khi lưu xong (`npc_director.go`).
- Ngày: `daily.go:33` dùng `AddDate` theo múi giờ máy chủ (DST); lượt đào qua nửa đêm không bị đếm (`account.go:509`).
- Lệch plan nhỏ: cameo nhà báo 2 phút (plan 8), việc dọn ổ 30 giây (plan 60), quà điểm danh khác plan, hoa không tặng được, giằng co 2 bậc hiếm 16/20 giây (plan 15/18), snapshot gửi mọi xe NPC (plan: trong 70 m), xe ôm không hoàn tiền khi máy chủ huỷ, cưỡi ngỗng khi kiệt sức, nghỉ ổ trong Tổ nhỏ không có tác dụng.
- Client: hai loại mã "#…" gây nhầm khi kết bạn; message bị bỏ do rate limit không báo gì (panel "Đang tải…" treo); giá xe ôm không tính mưa/mặc cả; lời mời thăm nhà hết hạn trước khi bấm vào; gợi ý thi công chỉ hiện sau khi mở bảng hợp đồng; nhiều action server không có UI (`dig_cancel`, `fish_cancel`, `rename_pet`…).
- Tài liệu: TIEN_DO/PLAN vẫn ghi "chưa làm" cho câu cá, lô trưng bày, tưới giúp, cấp 11–20, đèn pha; README/TIEN_DO ghi "69 test với -race" (thực tế 84, chưa chạy -race).

## Câu hỏi cần chốt (lượt 2)

1. Luống rau nhà: giới hạn theo hạng đất, trần bán ở quầy, hay giảm giá nông sản?
2. Xe liên trại: chạy theo lịch 3 phút như plan, hay sửa README và câu thoại theo cách hiện tại (20 giây sau người đầu)?
3. Cấp 11–20 có phải nội dung đã duyệt? Nếu có, cần cập nhật plan §12, §17.
4. PK Úp ngửa và thắng Gà Máy có tính bảng tuần, điểm hội, danh hiệu không?
5. Hết giờ chọn ở Úp ngửa: máy chủ chọn ngẫu nhiên hay xử vòng đó cho bên kia?
6. Thân thiết thú cưng có cần trần mỗi ngày không?
7. Mất mạng 1 giây giữa lượt đào thì mất lượt ngay (hiện tại), hay giữ lượt đến hết 15 giây chờ nối lại?

---

## Cập nhật: đồng bộ với plan (nhóm câu hỏi plan đã có đáp án)

| Mục | Plan | Đã làm |
|---|---|---|
| A1 xe liên trại | PLAN_LOGIC 7.3 | Lịch cố định `BusEvery` (live 3 phút); chỉ giữ ghế khi cửa mở `BusWait` (live 20 giây) trước giờ chạy; bảng bến luôn hiện chuyến kế, nút đặt vé tự bật khi cửa mở |
| A2 sức chứa nhà | 13.3 | `houseCapacity`: rơm 8, gỗ 12, gạch 16; `Visit` đọc qua chỉ mục `HomeStore.Capacity` (không chạm *Home của phòng) |
| A3 giao dịch vật phẩm | 11 | `TradeOffer.Items` theo cờ `ItemDef.Trade` (trứng, phiếu không); kho người nhận đầy thì hoàn nguyên cả hai |
| A4 gà con cũ | 15.2 | `normalize`: loài trống/lạ thành `ga_con` |
| A7, D3 xếp hạng PK | PLAN.md 10.1 | `Match.ranked()`: chỉ Mổ nhanh giữa hai người thật cộng `PKWins`, bảng tuần, điểm hội |
| Quà cấp khi kho đầy | PLAN.md 7.1 | `PendingItems` (tối đa 200), lệnh `pending_claim`, dòng "Quà chờ nhận" trong Túi đồ |
| `home_upgrade` | 13.2 | Bỏ lệnh và `UpgradeLand`; mua đất chỉ qua Ông Chủ Đầu Tư (`land_buy`) |
| Mất mạng giữa lượt đào | 4.3 | Giữ luật huỷ lượt; client (B4) dọn màn giằng co khi nhận `welcome` |

Test mới: `server/plan_alignment_regression_test.go` (7 test). Test balance đặt `BusEvery`/`BusWait` 1 giây.

Review đợt này: không lỗi nghiêm trọng. Đã sửa theo góp ý:
- Nhà đang xây/nâng cấp chỉ nhận 8 khách. `Capacity` xét thêm `DoneAt` vì `State` trong chỉ mục có thể cũ.
- Hàng chờ đầy thì báo số món bị bỏ (`Reward.Dropped`).
- Bảng bến chỉ chạy một vòng đếm; chuyến đã quá giờ quay về lịch.
- Đề nghị giao dịch không gửi lại món đã hết.
- Test PK ép người chơi thắng để luôn kiểm nhánh thắng Gà Máy.
- Test xe dùng chu kỳ 1000 giờ để không chập chờn theo đồng hồ thật.

Câu hỏi còn mở (đợt này):
- `PKWins` cũ kiếm từ Gà Máy/Úp ngửa có reset không? Hiện giữ nguyên.
- Hàng chờ quà 200 món có đủ, hay không giới hạn?
- Đang nâng cấp nhà thì sức chứa về 8 (hiện tại) hay giữ theo vật liệu cũ?

---

## Cập nhật: sửa nhóm mất tiền/mất đồ (M1, M2, M4–M6, M12–M14)

| # | Đã làm |
|---|---|
| M1 | `eat_worm` gọi `levelUp`: lên cấp do ăn nhận quà ngay |
| M2 | `uprootHomeBed`: luống về kho do đổi lô (`BuyLot`) hoặc xây nhà (`clearFoot`) thì chín thu, chưa chín trả hạt (+phân ủ); `home_pickup` chặn luống đang có cây |
| M4 | `txn` trả bool (`txnFunc`). `contract_create` lỗi lưu → `Discard`; `guild_create` lỗi → `Leave` giải thể; `land_buy` lưu tài khoản trong closure `withHome` trước khi lưu nhà, lỗi thì hoàn cả hai tại chỗ |
| M5 | `Work` lưu lỗi gỡ khoản công vừa ghi; nghiệm thu mà lưu nhà lỗi vẫn giữ chuồng (thợ đã được trả) |
| M6 | `GuildStore.Do` chụp hội, lỗi/lưu hỏng thì `rollbackLocked`; `UndoFund` (bù trừ) không rollback, lưu lỗi chỉ log |
| M12 | Điểm danh kho đầy: vật phẩm vào hàng chờ, không mất chuỗi ngày |
| M13 | `withHome`: CAS 0/1/2 + defer gửi kết quả (panic không treo); dựng lại `zones["home"]` sau khi đổi lô |
| M14 | `dig_finish`/`fish_finish` giữ phiên tới khi txn xong; lỗi thì `abandonDig`/`abandonFish` (vẫn tính hồi) |

Test mới: `server/save_failure_regression_test.go` (12 test; giả lập lưu hỏng bằng thư mục trùng tên file `.tmp`). Review: 2 lỗi do chính bản sửa (treo khi closure panic, `UndoFund` bị rollback) đã sửa. Mục "RID cũ đóng phiên đào né hồi" không xảy ra được vì `act` trả kết quả cũ trước mọi xử lý; có test giữ hành vi này.

Còn lại (thấp):
- Có thể có thợ nhận hợp đồng trong khoảng thời gian giữa `Create` và `Discard`, lúc đó `Discard` không chạy. Khoảng này chỉ dài bằng một lần lưu tài khoản. Cách triệt để: tạo hợp đồng ở trạng thái "pending".
- `Discard`/`Leave` mà lưu lỗi thì bản trên đĩa còn hợp đồng hoặc hội. Server khởi động lại thì chúng sống lại.
- Nghiệm thu mà lưu nhà lỗi thì không thử lưu lại.
- `land_buy` mà lưu nhà lỗi sau khi đã lưu tài khoản: người chơi bị báo lỗi dù đã trả tiền. Đất mới vẫn còn trong bộ nhớ.

---

## Cập nhật 06/11/2026: chốt luật và sửa phần còn lại

**Luật đã chốt (đều theo đề xuất):**
- Luống rau nhà theo lô: 2/4/6/8, giới hạn cả lúc đặt lẫn số luống đang trồng.
- Câu cá 30 con/ngày.
- Thân thiết từ bạn bè tối đa 60/ngày.
- Thắng vì đối thủ bỏ trận không tính xếp hạng.
- Kho 200 tính chung.
- Khách nghỉ ổ hồi như chủ (code vốn đã vậy).
- Giữ cấp 11–20, cập nhật plan.
- Giữ nguyên số trận thắng cũ.

**Server:**
- **M7:** đề nghị chuyển quyền chủ hội có hạn 24 giờ và bị xoá khi người được đề nghị rời hội hoặc bị mời ra. Lời mời vào hội cũng hạn 24 giờ, kèm kiểm tra chặn.
- **M8:** đăng nhập cập nhật lần có mặt (`Touch`), lưu nếu cũ hơn 1 giờ.
- **M9:** khu đã đủ người thì gà dừng ở cổng. Mới chặn khi đi bộ; xuống xe vào khu đầy vẫn lọt.
- **M10:** hết đếm ngược mà một bên vẫn mất kết nối thì huỷ trận (nối lại kịp thì vẫn đấu). Kết quả trận chốt trong lúc mất mạng được gửi lại khi nối lại (`pkEnd`).
- **M11:** danh hiệu tuần chưa nhận được giữ trong `Owed` qua nhiều tuần.
- **Lỗi thấp:**
  - Kích thước giun tính theo bậc hiếm.
  - `DoWait` trả về ngay khi phòng đã dừng.
  - Giới hạn đọc message tăng lên 16 KB.
  - Lô trưng bày lọc nhà riêng tư và nhà của người đã chặn.
  - Khách xuống xe ở ô đi được.
  - Mỗi người chỉ gọi một xe ôm.
  - Cứu đói tính cả nông sản/cá, tối đa 3 lần/ngày (đếm khi làm xong).
  - `restore` giữ sổ cái của thay đổi trước đó.
  - Xe liên trại đã xuất phát thì khách không xuống được nữa.
  - Lên xe buýt/xe ôm chỉ sau khi vé đã lưu.
  - Mốc ngày điểm danh không phụ thuộc giờ mùa hè của máy chủ; lượt đào qua nửa đêm được tính cho ngày mới.
  - Giao dịch tôn trọng giới hạn túi 60 giun, và huỷ khi hai bên đi xa nhau quá 8 m.
  - Mời PK lại cùng một người phải cách 1 phút; đấu Gà Máy thì huỷ lời mời đang chờ.
  - Thắng cùng một đối thủ tối đa 5 trận/tuần được tính.
- **Lệch plan:**
  - Cameo 8 phút, dọn ổ 60 giây.
  - Quà điểm danh theo plan §16.
  - Giằng co 15/18 giây (tỉ lệ bắt vẫn đạt bảng §4.2).
  - Không cưỡi ngỗng khi đói lâu.
  - Tổ nhỏ chưa có nhà: nghỉ ổ tụt độ no chậm một nửa.
  - Tặng bó hoa cho bạn: +5 thân thiết, tính trong trần 60.
- **Client (agent):** sửa B5, B7, B8, B10–B13, L14; thêm nút "Bỏ" lượt đào/câu, đổi tên thú cưng, chống treo "Đang tải…", ghi rõ "Mã bạn bè", xử lý `pk_end` đến muộn, nút tặng hoa, số luống trong bảng mua đất. Server gửi thêm `sellPrices` và `now`.

**Không sửa (có lý do):**
- Snapshot vẫn gửi vị trí mọi xe NPC, vì bản đồ và minimap cần biết xe buýt ở đâu.
- Xe ôm không hoàn tiền: không có đường nào để máy chủ huỷ chuyến đang chở. Người chơi tự xuống hoặc mất mạng thì theo plan không hoàn.
- Bỏ `fsync`: mỗi giao dịch đều ghi trên goroutine phòng, và trên Windows fsync làm giật tick.
- `rid` rỗng: client luôn gửi `rid`. Nếu thiếu thì chỉ mất tính idempotent, không cộng tiền hai lần.
- "Phát lại RID cũ để đi xe hoặc đóng lượt đào" không xảy ra được, vì `act` trả kết quả cũ trước mọi xử lý.

Test mới: `server/rules_and_low_fixes_regression_test.go` (15 test). Tổng cộng 122 test.

**Bổ sung 06/11/2026:** giới hạn người trong khu đã chặn cả xuống xe và đổi kênh vào khu đầy (đặt ra ngoài cổng), thêm hàng chờ ở cổng khu (`server/area_capacity_queue.go`, 5 test mới, tổng 127).
