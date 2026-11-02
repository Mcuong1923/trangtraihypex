# Hypex Farm — Plan logic game v2: làng sống, di chuyển, đào giun, nhà và thú cưng

Ngày: 02/11/2026 · Phiên bản 1.1 · Trạng thái: **đã triển khai P0–P7** (xem mục "Trạng thái triển khai" ngay dưới; tiến độ chi tiết ở [TIEN_DO.md](TIEN_DO.md)).

Plan này biến các yêu cầu chơi thử của chủ dự án thành luật cụ thể có số liệu, thay đổi mã theo module, giao thức mạng, kiểm thử và lộ trình. Nó bổ sung cho [PLAN.md](PLAN.md) (luật gốc), [PLAN_THE_GIOI_3D.md](PLAN_THE_GIOI_3D.md) (bản đồ liền) và [PLAN_GIAO_DIEN_BO_2.md](PLAN_GIAO_DIEN_BO_2.md) (mỹ thuật). Chỗ nào plan này đổi số liệu của PLAN.md, khi triển khai phải sửa luôn PLAN.md để chỉ còn một nguồn đúng.

## Trạng thái triển khai (02/11/2026)

| # | Yêu cầu | Trạng thái | Mã chính |
| --- | --- | --- | --- |
| 1 | Độ no 15–20 phút | ✅ bộ `live` là mặc định: 4,5 điểm/phút + 0,4/lượt đào | `config.go` |
| 2 | 100 thóc khởi nghiệp, hết năng lượng vẫn đào | ✅ quà một lần; kiệt sức đào được giun thường/khá hiếm; ăn sống; cám cứu trợ | `account.go`, `dig.go` |
| 3 | Nút chạy nhanh, nghỉ ~5 giây | ✅ ×1,65 trong 6 giây, hụt hơi khoá 5 giây; nút 🏃 và Shift | `room.go`, `main.js` |
| 4 | Xe buýt NPC chạy liên tục, 2 lượt miễn phí, hồi 5 phút | ✅ 3 xe, 9 bến, 3 thóc khi hết lượt, chip 🎫 trên HUD | `transit.go`, `npc.go`, `npc.js` |
| 5 | Cụ Cục chào gà mới bằng tên, bong bóng thoại | ✅ kèm dặn luật 4 trang, nút "Tôi biết chơi rồi" | `npc_director.go`, `panels.js` |
| 6 | NPC chủ động chào, chủ tiệm xe giảm giá | ✅ phiếu giảm 10% xe đầu tiên, lời chào theo trạng thái và theo buổi | `npc_director.go` |
| 7 | Xe ôm mời đi, chọn điểm trên bản đồ, 1–2 thóc | ✅ số xe theo người ngoài trời, mời/từ chối có hạn mức | `npc.go`, `map.js` |
| 8 | NPC đấu nhau, Chó Mực trọng tài, khán giả bàn tán | ✅ ở Sân tỷ thí; ngày hội thì ra sân khấu Ao và vườn | `npc_director.go`, `npc.js` |
| 9 | Xe đạp/xe máy/ván trượt, thiếu gia ô tô, hai mẹ con | ✅ cùng shipper hỏi đường, nhà báo, bán rong, cụ dưỡng sinh | `npc.go` |
| 10 | Hướng dẫn qua NPC, hài hước | ✅ 6 việc có dấu "!" + mũi tên, tự trao thưởng | `npc_director.go` |
| 11 | Cây che tầm nhìn | ✅ cây chắn camera mờ dần | `scenery.js` |
| 12 | Đào: mổ → popup → cận cảnh → giữ để giằng co | ✅ máy chủ chạy lại mô phỏng; `tug.js` khớp 1-1 (test Node) | `dig.go`, `dig.js`, `tug.js` |
| 13 | Chủ đầu tư bán đất, chọn vật liệu xây nhà | ✅ 3 vị trí, 2 thiết kế × rơm/gỗ/gạch, nâng cấp | `estate.go`, `life.js` |
| 14 | Lên cấp tặng thóc, vật liệu, thức ăn, hạt, trứng | ✅ bảng cấp 2–10, kho 200 món | `items.go`, `daily.go` |
| 15 | Trứng nở thú cưng có kỹ năng | ✅ 3 loại trứng, 7 loài/7 kỹ năng, giới hạn cộng dồn | `pets.go`, `v11.js` |
| 16 | Logic bổ sung | ✅ điểm danh 7 ngày, nghỉ ổ (cả cạnh bạn bè), ngày/đêm, mưa, nhiệm vụ NPC, xếp hạng tuần | `clock.go`, `weekly.go`, `community.go` |

**Khác đặc tả hoặc để sau:**
- Luống cộng đồng làm theo kiểu "mỗi người một lớp": ai cũng trồng được ở 12 luống, mỗi người giữ tối đa 1, chỉ thấy cây của mình. Lý do giống nhà riêng: không có chuyện hết luống hay giành luống.
- Chưa làm: tưới giúp luống bạn bè (mục 14), lô trưng bày nhà người chơi ở Xóm (P5b, mục 13.2), đèn pha xe NPC ban đêm (mục 9.1), câu cá và cấp 11–20 (mục 16, ưu tiên thấp).
- Xe buýt luôn 3 xe thay vì 2/3 theo số người: mỗi bến có xe khoảng 1 phút một chuyến.
- Bảng xếp hạng tuần chỉ tính PK với người thật (không tính Gà Máy) để không cày điểm.

## 0. Yêu cầu → mục trong plan

| # | Yêu cầu | Mục |
| --- | --- | --- |
| 1 | Độ no (mana) hết quá nhanh, kéo dài ra khoảng 15–20 phút | 2 |
| 2 | Hết độ no thì không đào được, không có thóc: cấp 100 thóc khi vào game; hết năng lượng vẫn đào được | 2.3, 4.6 |
| 3 | Đi chậm: thêm nút chạy nhanh, chạy một lúc phải nghỉ khoảng 5 giây | 3 |
| 4 | Xe buýt do NPC lái chạy liên tục qua các bến; 2 lượt miễn phí, hết thì trả thóc hoặc đợi khoảng 5 phút hồi lượt | 7 |
| 5 | Cụ Cục chào gà mới bằng tên và dạy luật; NPC nói bằng bong bóng trên đầu | 5, 6 |
| 6 | NPC chủ động bắt chuyện, ví dụ chủ cửa hàng xe chào và giảm giá | 9.4 |
| 7 | Xe ôm NPC dọc đường mời đi; người chơi chọn đi hay không; 1–2 thóc tuỳ nơi đến; số xe ôm vừa đủ; chọn điểm đến trên bản đồ, NPC chở tới | 8 |
| 8 | NPC tự tương tác: 2 gà NPC đấu, Chó Mực làm trọng tài, khán giả bàn tán, cổ vũ, cười nhạo | 9.3 |
| 9 | NPC đi xe đạp, xe máy, ván trượt trên đường; thiếu gia đi ô tô trêu người đi bộ; hai mẹ con gà đi qua bình luận | 9.1, 9.2 |
| 10 | Hướng dẫn người mới qua NPC, dễ hiểu và hài hước | 6 |
| 11 | Đi qua sau cây bị che tầm nhìn | 10 |
| 12 | Đào giun: mổ để đào → popup giun xuất hiện → góc nhìn gần, dùng mỏ kéo → nút giữ để giằng co; giun hiếm giằng co lâu, có thể chạy thoát | 4 |
| 13 | Chủ đầu tư bán đất gần khu nhà; mua đất, chọn vật liệu xây nhà | 13 |
| 14 | Lên cấp tặng thóc và vật phẩm: vật liệu, thức ăn, hạt giống, trứng | 11, 12, 14 |
| 15 | Trứng nở ra bạn đồng hành, mỗi loài một khả năng (may mắn khi đào, giảm giá xe, cõng chạy nhanh…) | 15 |
| 16 | Tự bổ sung và hoàn thiện các logic khác | 16 |

## 1. Nguyên tắc chung

1. **Máy chủ quyết định mọi thứ có giá trị:** độ no, thể lực, tiền, vé, kết quả giằng co giun, đường đi của xe NPC, ai được mời xe ôm. Client chỉ gửi ý định và hiển thị, có thể dự đoán để mượt nhưng máy chủ chốt.
2. **Mọi thay đổi tài sản giữ kiểu hiện có:** có request ID chống xử lý lặp, ghi sổ cái thóc, lưu đĩa trước khi báo thành công, lỗi thì hoàn nguyên (xem `txn` trong `server/actions.go`).
3. **NPC không bao giờ giả làm người chơi:** tên NPC có biểu tượng riêng, không tính vào số người online, không có trang hồ sơ, không nhận tin nhắn riêng.
4. **Số liệu cân bằng ở một chỗ:** thêm bộ cân bằng mới `live` làm mặc định (mục 17). `demo` giữ để thử siêu nhanh, `plan` giữ số của PLAN.md cho đối chiếu. Server khởi động bằng `-balance live` khi chạy thật và chạy LAN.
5. **Không có ngõ cụt:** người chơi luôn có ít nhất một cách hồi độ no và kiếm thóc mà không cần thóc, kể cả khi kiệt sức (mục 2.3).
6. **Mỗi luật mới phải có test** ở mức hàm thuần (`*_test.go`) và ít nhất một e2e qua WebSocket cho luồng chính.

## 2. Độ no (mana) và chống kẹt

### 2.1. Hiện trạng trong mã

- `server/config.go`: `demo` trừ **0,5 điểm/giây** (100 điểm hết sau ~3 phút 20 giây); `plan` trừ 1 điểm/phút. Mỗi lượt đào trừ thêm 1 điểm.
- `server/account.go` `HungerStage`: dưới 20 là "sắp đói"; ở 0 lâu `HungerSlowSec` thì đi chậm 15%, lâu `HungerExhaustSec` thì kiệt sức. `demo`: 20 giây và 45 giây.
- `CanDig` chặn đào khi kiệt sức (`stage >= 4`). Tài khoản mới có **0 thóc** (`NewAccount`). Thưởng hướng dẫn 100 thóc chỉ nhận sau khi xong 5 bước.
- Việc cứu đói (`RescueEligible`) chỉ mở khi độ no ≤ 20, ví dưới 5 thóc và không còn gì ăn hoặc bán. Đúng về luật nhưng người chơi khó tìm ra.

### 2.2. Luật mới (bộ `live`)

| Thông số | Cũ (`demo`) | Mới |
| --- | --- | --- |
| Tụt khi online | 0,5/giây | **4,5/phút** (0,075/giây) |
| Tụt khi ngồi ổ (sảnh hoặc nhà riêng) | như trên | **giảm một nửa** (2,25/phút) |
| Trừ thêm mỗi lượt đào | 1 | **0,4** |
| Chạy nhanh | — | không tốn độ no, chỉ tốn thể lực (mục 3) |
| Sắp đói | < 20 | giữ nguyên |
| Đi chậm 15%, không chạy nhanh | sau 20 giây ở 0 | **sau 3 phút ở 0** |
| Kiệt sức: đi chậm 25%, không lái xe, không PK; **vẫn đào được** kiểu "đào tay không" (mục 4.6) | sau 45 giây ở 0, không đào được | **sau 6 phút ở 0** |

Thời gian từ đầy tới 0: đi dạo khoảng **22 phút**, đào liên tục (~5 lượt/phút) khoảng **15 phút**, chơi xen kẽ khoảng **18 phút**, ngồi ổ nghỉ khoảng 44 phút. Đúng khoảng 15–20 phút đã yêu cầu.

Thức ăn giữ giá: cám 5 thóc +20, rau 12 thóc +50, suất 20 thóc +100. Với tốc độ mới, một giờ chơi tốn khoảng 270 độ no, tức 50–60 thóc nếu mua suất ăn. Riêng đào giun đã kiếm 9–10 thóc/lượt theo kỳ vọng nên không bị lỗ.

### 2.3. Không bao giờ kẹt

1. **Quà khởi nghiệp 100 thóc:** tài khoản mới tạo ra có sẵn 100 thóc, ghi sổ cái lý do `starter_gift`. Tài khoản cũ nhận đúng một lần ở lần đăng nhập đầu sau khi cập nhật (cờ `StarterGranted`).
2. **Thưởng hướng dẫn đổi thành:** 50 thóc + 3 khẩu phần cám + 1 trứng thường. Tổng thóc cho người mới vẫn khoảng 150; thêm trứng để làm quen hệ thú cưng sớm.
3. **Ăn sống giun:** giun trong túi ăn được ngay mà không cần trả phí chế biến. Hồi 60% độ no của món chế biến (giun đất +6), không cộng XP. Gà ăn giun là chuyện thường, nên người chơi tự nghĩ ra được cách này.
4. **Đào khi kiệt sức:** xem mục 4.6. Người chơi 0 thóc, 0 độ no vẫn đào được giun thường, rồi ăn sống hoặc bán.
5. **Bữa cứu trợ của Cụ Cục:** mỗi ngày một lần, nếu độ no < 10, ví < 5 thóc và không có gì ăn, nói chuyện với Cụ Cục sẽ nhận 1 khẩu phần cám kèm lời nhắc đi đào giun. Việc cứu đói "dọn ổ rơm" giữ nguyên làm lối thứ ba.
6. **Cảnh báo sớm:** khi xuống dưới 20, NPC gần nhất nhắc bằng bong bóng (mục 9.5) và HUD nhấp nháy thanh độ no.

### 2.4. Thay đổi mã

- `config.go`: thêm `LiveBalance()`, trường `ManaRestFactor`, `RawEatRatio`, `StarterThoc`; đổi `HungerSlowSec`/`HungerExhaustSec`.
- `account.go`: `NewAccount` cộng 100 thóc; `normalize` cấp quà một lần cho tài khoản cũ; `TickVitals(dt, b, resting)`; `EatWormRaw`; `CanDig` bỏ chặn kiệt sức (chuyển thành chế độ đào tay không).
- `room.go`: truyền `resting` khi người chơi đang ngồi ổ (`p.nest != ""`).
- `main.go`: thêm `-balance live` và đặt làm mặc định.
- Client `panels.js`: nút "Ăn sống" trong túi; hiển thị thời gian còn lại trước khi đói ("còn khoảng 12 phút").

### 2.5. Nghiệm thu

- Test hàm: 22 phút đi dạo từ 100 xuống ~1; 15 phút đào 5 lượt/phút xuống ~0; ngồi ổ chậm đúng một nửa.
- Tài khoản mới có 100 thóc; tài khoản cũ nhận đúng một lần (đăng nhập lại không nhận thêm).
- Kịch bản e2e "0 thóc, 0 độ no, túi trống, đã kiệt sức": đào được giun → ăn sống → độ no tăng; không có thao tác nào báo lỗi vô lối thoát.

## 3. Chạy nhanh

### 3.1. Luật

| Thông số | Giá trị |
| --- | --- |
| Tốc độ đi bộ | 4,2 → **4,6 m/giây** (bản đồ rộng hơn trước) |
| Tốc độ chạy | **×1,65** = 7,6 m/giây |
| Thể lực | 100 điểm = **6 giây** chạy liên tục |
| Hụt hơi | thể lực về 0 thì **khoá chạy 5 giây**; hết 5 giây thể lực đầy lại |
| Hồi khi thả nút sớm | sau 0,8 giây nghỉ, hồi 25 điểm/giây (đầy sau 4 giây) |
| Đói (giai đoạn 2) | thể lực tối đa còn một nửa (3 giây) |
| Đi chậm / kiệt sức (giai đoạn 3–4) | không chạy được |
| Trên xe, đang đào, đang ngồi ổ, đang giao dịch | không chạy |
| Bạn đồng hành "Ngỗng Lưng Chắc" | +2/+3/+4 giây thể lực (mục 15) |

### 3.2. Máy chủ

- Gói `in` thêm cờ `run` (bool). `Player` thêm `stam float64`, `stamLockUntil time.Time`, `running bool` (không lưu đĩa, thuộc phiên).
- `room.step`: nếu `run` và đủ điều kiện thì nhân tốc độ, trừ thể lực theo `dt`; về 0 thì đặt khoá 5 giây. Gửi thể lực cho chủ phiên trong gói `vit` (đã gửi mỗi giây; khi đang chạy tăng lên 5 lần/giây). Snapshot thêm cờ `r` để người khác thấy dáng chạy.
- Tốc độ vẫn do máy chủ tính, nên client sửa cờ cũng không chạy nhanh hơn được.

### 3.3. Client

- Desktop: giữ **Shift** để chạy. Điện thoại: nút **🏃 Chạy** to cạnh joystick, giữ để chạy, có vòng thể lực quanh nút. Khi bị khoá, vòng đếm ngược 5 giây kèm chữ "Hụt hơi…".
- Hoạt ảnh: chân đánh nhanh gấp đôi, thân nghiêng tới, bụi sau chân, cánh xoè nhẹ. Hụt hơi: gà chống cánh lên gối thở 1 giây (chỉ là hoạt ảnh, không khoá di chuyển).
- Dòng hướng dẫn HUD đổi thành "WASD đi · Shift chạy · E tương tác · M bản đồ".

### 3.4. Nghiệm thu

Test hàm cho ngân sách 6 giây, khoá 5 giây, hồi sớm và giới hạn khi đói; e2e đo quãng đường 2 giây đi và 2 giây chạy (tỉ lệ ~1,65). Gói `in` có `run` nhưng đang kiệt sức thì không nhanh hơn.

## 4. Đào giun v2: mổ → giun ló → cận cảnh → giằng co

### 4.1. Luồng trải nghiệm

1. **Lại gần ụ đất, bấm E.** Gà cúi mổ đất 3 nhịp (0,9 giây), đất bắn lên, ụ rung. Người xung quanh cũng thấy.
2. **Giun ló đầu.** Popup nhỏ trồi lên trên ụ: "Có gì đó ngọ nguậy…!" rồi hiện giun thật cùng màu viền độ hiếm (thường xám, khá hiếm xanh, hiếm tím, sử thi vàng, huyền thoại cầu vồng). Giun hiếm có hiệu ứng lấp lánh và âm báo riêng.
3. **Camera cận cảnh** (0,4 giây chuyển cảnh): máy đặt thấp sát mặt đất, nhìn chéo vào mỏ gà và miệng hố. Gà ngậm đầu giun; thân giun co giãn theo lực kéo.
4. **Giằng co:** nút lớn **"GIỮ ĐỂ KÉO"** ở giữa dưới (giữ Space, giữ chuột hoặc giữ ngón tay). Có hai thanh: **Tiến độ** (kéo được bao nhiêu) và **Độ căng** (xanh → vàng → đỏ). Giun giật theo từng cơn; giữ khi giun đang giật mạnh thì độ căng vọt lên. Tới đỏ là giun **tuột** lại một đoạn.
5. **Kết quả:**
   - Kéo đủ: gà ngửa cổ giơ giun lên, thẻ kết quả hiện loài, chiều dài, độ hiếm và kỷ lục.
   - Giun thoát: giun chui tọt vào hố, bụi bay, kèm một câu an ủi hài (mục 9.5).
   - Camera trở lại góc thường.

### 4.2. Mô hình giằng co (máy chủ và client chạy cùng công thức)

Mỗi lượt có `seed` do máy chủ tạo. Từ seed sinh lịch các cơn giật: thời điểm, độ dài 0,4–1,2 giây, lực. Mỗi bước 50 ms:

- Đang giữ: `tiến_độ += tốc_kéo × (1 − độ_căng)`; `độ_căng += lực_giật_hiện_tại × hệ_số_căng`, ngoài cơn giật thì độ căng tăng chậm.
- Đang thả: độ căng giảm nhanh (−1,5/giây); tiến độ tụt chậm (giun rút lại) theo `tốc_rút`.
- Độ căng ≥ 1: **tuột**, tiến độ −25%, độ căng về 0,3, đếm 1 lần tuột.
- Tiến độ ≥ 100%: bắt được. Quá giờ, hoặc **tuột 3 lần** với giun từ "Hiếm" trở lên: giun thoát.

| Độ hiếm | Giờ tối đa | Tốc kéo | Cơn giật | Có thể thoát | Tỉ lệ bắt được (ước tính) |
| --- | --- | --- | --- | --- | --- |
| Thường | 6 giây | nhanh | yếu, thưa | không (tuột chỉ làm chậm) | 100% |
| Khá hiếm | 8 giây | vừa | vừa | không | 100% |
| Hiếm | 12 giây | vừa | mạnh, dày hơn | có | 85–95% |
| Sử thi | 15 giây | chậm | mạnh, có cơn kép | có | 70–90% |
| Huyền thoại | 18 giây | chậm | rất mạnh, đổi nhịp | có | 55–80% |

Thời gian một lượt bắt giun thường: khoảng 2–3 giây. Các hệ số cụ thể chốt bằng chơi thử; test mô phỏng phải chứng minh "người chơi hợp lý" (thả khi độ căng > 0,7, giữ khi < 0,4) đạt đúng các khoảng tỉ lệ trên.

### 4.3. Máy chủ quyết định, chống gian lận

- `dig_start` (giữ tên): kiểm tra khoảng cách, túi, hồi đất như cũ. Máy chủ **gieo loài giun ngay lúc này** (giữ bảng xác suất và giun sự kiện như cũ), tạo `seed`, trả về `{id, worm, rarity, seed, params, peckMs}`. Lộ trước loài giun là chủ ý, để màn giằng co hồi hộp ("Giun vàng! Giữ chặt!").
- Client chạy mô phỏng tại chỗ để phản hồi tức thì, ghi lại dòng thời gian giữ/thả (tối đa 300 sự kiện).
- `dig_finish {id, events:[[ms, giữ/thả], …]}` thay cho `dig_stop`. Máy chủ:
  - kẹp thời gian trong khoảng đồng hồ của mình, dung sai trễ mạng 0,6 giây như hiện có;
  - chạy lại cùng mô phỏng với seed;
  - chốt kết quả; nếu lệch với client thì máy chủ thắng.
- Giới hạn: không gửi `dig_finish` sớm hơn thời gian kéo tối thiểu; chỉ một lượt đào mở mỗi người; lượt bỏ ngang hết hạn sau 25 giây; rời khu, lên xe hoặc mất kết nối thì huỷ lượt.
- Rủi ro chấp nhận: tool tự động có thể chơi giằng co hoàn hảo. Lợi thế chỉ là tỉ lệ bắt giun hiếm cao hơn, vẫn bị giới hạn 120 lượt đào có thưởng mỗi ngày và hồi đất như người thường. Log tỉ lệ bắt theo tài khoản để quản trị soi bất thường.

### 4.4. Kinh tế

- Giun thoát làm giảm thu nhập kỳ vọng ở nhóm hiếm. Bù bằng cách tăng nhẹ tỉ lệ gốc: Hiếm 4% → 4,6%, Sử thi 0,9% → 1,05%, Huyền thoại 0,1% → 0,13%; Giun đất giảm tương ứng để tổng vẫn bằng 1. Người chơi trung bình giữ nguyên thu nhập, người khéo tay được thêm một chút.
- Giun thoát: **không tính vào 120 lượt đào có thưởng của ngày**, vẫn tính hồi đất, +5 XP an ủi.
- **Đào cũng cho XP:** +2 / 4 / 8 / 15 / 25 XP theo độ hiếm (nhỏ hơn XP khi ăn, vốn là nguồn chính). Như vậy đúng ý "đào giun để lên cấp".
- Hồi đất bộ `live`: **8 giây** sau mỗi lượt (đã gồm thời gian mổ và kéo, nhịp thực khoảng 5 lượt/phút).
- Bỏ thanh canh lực "perfect" cũ: màn giằng co thay thế nó. Thống kê `Perfect` đổi nghĩa thành "kéo không tuột lần nào".

### 4.5. Client

- `client/js/panels.js` `startDig`: viết lại thành bộ máy trạng thái `peck → reveal → tug → result`. Mô phỏng giằng co tách vào `client/js/dig.js`, cùng công thức với máy chủ. Mã nguồn công thức có test chung: bảng giá trị mẫu xuất từ Go, client so khớp.
- Camera: `CameraController` thêm chế độ `closeup(target, mound)` đặt máy cách mỏ ~1,2 m, cao 0,6 m, FOV 50; trả lại chế độ trước khi xong.
- Mô hình giun: ống cong nhiều đốt (`TubeGeometry`) nối mỏ và miệng hố, co giãn theo độ căng; giun hiếm có vân phát sáng.
- Âm thanh: tiếng mổ đất, tiếng "dây" căng, "bụp" khi tuột, nhạc ngắn khi bắt được giun hiếm.
- Trợ năng: chế độ "chạm để kéo" (mỗi lần chạm kéo một nấc, không cần giữ) cho người khó giữ nút lâu; "giảm chuyển động" thì bỏ rung camera.

### 4.6. Đào khi kiệt sức ("đào tay không")

- Được đào ở mọi giai đoạn đói. Ở **giai đoạn 4** (kiệt sức): mổ chậm (1,4 giây), hồi đất ×1,5, bảng giun **chỉ còn hai bậc đầu** (Thường 85%, Khá hiếm 15%). XP và các luật khác giữ nguyên.
- Thông báo rõ: "Kiệt sức: chỉ đào được giun thường. Ăn sống một con để lấy sức nhé!"

### 4.7. Nghiệm thu

- Test mô phỏng: 1.000 lượt mỗi bậc với "người chơi hợp lý" nằm trong khoảng tỉ lệ bắt ở 4.2; "thả mặc" (không bấm) không bao giờ bắt được giun hiếm; giun thường luôn bắt được.
- Máy chủ và client cho cùng kết quả trên 200 dòng thời gian mẫu.
- e2e: bắt được thì vào túi, thoát thì không, và không tính vào hạn ngày; gửi `dig_finish` quá sớm bị từ chối.

## 5. Nền tảng NPC sống

### 5.1. Kiến trúc

**Hai loại NPC:**

1. **NPC cố định:** Cụ Cục, Lợn Ủn, Vịt Bông, Chim Chíp, Dê Thợ Máy, Bò Mộc, Chó Mực, Chủ đầu tư… Đã có vị trí trong `world.go` (kind `npc_*`, `shop_*`, `pk`). Thêm **lời chào theo khoảng cách** và **hội thoại khi bấm E**.
2. **Diễn viên (actor) di động:** xe buýt, xe ôm, xe đạp, xe máy, ván trượt, người đi bộ, thiếu gia ô tô, hai mẹ con gà, đấu sĩ ở sân. Máy chủ giữ vị trí và trạng thái.

**Máy chủ (file mới `server/npc.go`, `server/npc_lines.go`, `server/roads.go`):**

- `type Actor struct { ID, Kind, Name, Ride string; Pos Vec; RY, Speed float64; Path []Vec; PathI int; State string; StateAt time.Time; Seats []string; Target string; Script *Script }`.
- `Room.actors map[string]*Actor`; `Room.stepActors(now, dt)` gọi trong `step`. Mỗi kiểu có một bộ máy trạng thái riêng (mục 7, 8, 9).
- **Đạo diễn** (`director`) mỗi phòng, chạy 2 lần/giây: xét điều kiện từng người chơi để kích hoạt lời chào, lời mời xe ôm, cameo, câu nhắc, theo hạn mức ở 5.2.
- Snapshot thêm mảng `a` gồm actor trong bán kính 70 m quanh người nhận: id, loại, tên, x, z, ry, trạng thái, xe, ghế.
- Gói `say {id, text, ttl, to?, mood?}` gửi cho người trong bán kính 25 m (NPC cố định trong khu) hoặc 35 m (actor ngoài đường).
- **Đường xá phía máy chủ:** hiện đường làng do client tự vẽ từ quảng trường tới cổng các khu. Chuyển đồ thị đường (vòng quảng trường + đường nhánh + **đường vành đai mới** nối cổng các khu theo vòng tròn) sang `roads.go`, gửi kèm `WorldInfo`. Client vẽ đúng các đoạn đó, nên xe NPC chạy trên đúng con đường người chơi nhìn thấy.

**Client (file mới `client/js/npc.js`):**

- Dựng và nội suy actor như người chơi khác (`addPlayer`/`updatePlayers`). Mô hình có sẵn: `buildBus`, `buildVehicle`, `buildChicken` + trang phục. Thêm ván trượt và ô tô thiếu gia (sơn đỏ bóng, kính râm).
- **Bong bóng NPC** dùng `bubble()` sẵn có, kiểu riêng: nền kem viền nâu, tên NPC in nhỏ phía trên, đuôi bong bóng chỉ vào miệng. Câu dài tự xuống dòng (tối đa 2 dòng, ~70 ký tự); câu dài hơn chia thành nhiều bong bóng nối tiếp. Câu nói **với mình** (`to == pid`) có viền cam và đồng thời ghi vào nhật ký chat với nhãn 🐔 NPC.
- Ưu tiên: mỗi NPC chỉ một bong bóng một lúc; trên màn hình tối đa 4 bong bóng NPC, bỏ bớt câu ít quan trọng (khán giả) trước.

### 5.2. Tần suất, giọng điệu, cài đặt

| Loại | Hạn mức |
| --- | --- |
| Chào khi lại gần NPC cố định | mỗi NPC chào mỗi người 1 lần/ngày bằng câu đầy đủ; quay lại sau ≥ 10 phút thì chào ngắn |
| Câu trêu (thiếu gia, mẹ con…) | mỗi người ≤ 1 lần/5 phút, ≤ 6 lần/giờ; không trêu trong 5 phút đầu của người mới |
| Cameo nhắm vào một người | mỗi người ≤ 1 lần/8 phút |
| Mời xe ôm | xem mục 8 |
| Toàn phòng | ≤ 3 câu NPC/giây; khán giả tối đa 1 câu/giây |

**Giọng điệu:** hài, trêu nhẹ kiểu làng quê, **chỉ đùa về việc trong game** (đi bộ, đói, thua PK, đào hụt giun). Không bàn về ngoại hình, giới tính, tên thật hay đời thực. Không dùng từ tục. Tên người chơi chỉ chèn qua `{ten}`; tên đã qua kiểm duyệt ở `ValidateName`. Câu nhắc tới người chơi bằng ngôi thứ ba chọn theo giới tính hồ sơ: "anh kia" / "chị kia" / "bạn kia" (giới tính "khác" hoặc "ẩn" dùng "bạn kia").

**Cài đặt người chơi** (lưu trong `Account.Prefs`):

- "NPC trò chuyện": Đầy đủ / Vừa / Ít. Mức Ít chỉ giữ hướng dẫn, mời xe và lời của chủ quán.
- "Cho NPC trêu mình": bật (mặc định) / tắt.

### 5.3. Thư viện lời thoại

Lời thoại là dữ liệu, không viết cứng trong logic: `server/npc_lines.go` (bảng Go, sau có thể tách JSON). Mỗi mục gồm khoá sự kiện, điều kiện, trọng số và danh sách biến thể. Máy chủ chọn câu theo seed; client chỉ hiện.

```go
{Key: "rich.mock_walker", When: "player.onFoot && !player.ownsVehicle", Weight: 3, Lines: []string{
  "Ơ kìa, thời buổi này còn có gà đi bộ à? Mua xe đi chứ cưng!",
  "Đi bộ cho thon đùi hả {ten}? Để anh chở… à thôi, xe anh không chở gà đi bộ!",
}}
```

Mỗi NPC cố định cần ít nhất: chào lần đầu, chào lại, chào khi người chơi đói, chúc mừng lên cấp, phản ứng khi người chơi có giun hiếm, và 3 câu tán gẫu. Ví dụ ở mục 6 và 9.

## 6. Hướng dẫn người mới qua NPC

### 6.1. Kịch bản (thay danh sách 5 bước khô hiện tại)

| Bước | Ai nói | Việc cần làm | Ví dụ lời |
| --- | --- | --- | --- |
| 1 | Cụ Cục gọi từ bục ngay khi gà mới vào sảnh | Lại gần, bấm E | "Ơ kìa, gà mới! {ten} đấy hả? Chào mừng tới Trại Hypex — nơi giun béo, thóc đầy, và chưa ai bị làm gà rán. Lại đây cụ dặn mấy câu!" |
| 2 | Cụ Cục (hội thoại 4 trang ngắn) | Đọc luật | "Luật số một: bụng đói thì mổ chậm. Thanh **Độ no** tụt dần, hết là chân run như cầy sấy." · "Luật số hai: đói thì ăn, nghèo thì đào, lười thì… vẫn phải đào." · "Cụ cho {ten} 100 thóc làm vốn. Tiêu cho khéo, đừng mua hết mũ!" |
| 3 | Gà Đón Khách ở cửa | Giữ Shift / nút 🏃 chạy ra cửa | "Thử chạy coi! Chạy lâu hụt hơi thì nghỉ 5 giây — gà chứ có phải siêu nhân đâu." |
| 4 | Biển chỉ đường + Chim Chíp ở bến cửa sảnh | Đi bộ hoặc **đi xe buýt miễn phí** tới Vườn giun | "Lên xe đi bà con! Lượt đầu miễn phí, lượt sau cũng… miễn phí, tới lượt thứ ba mới tính tiền!" |
| 5 | Lợn Ủn ở vườn | Đào con giun đầu tiên (màn giằng co có chữ hướng dẫn to) | "Lần đầu đào hả? Mổ nhẹ thôi kẻo giun nó kiện! Giữ nút để kéo, thấy đỏ thì thả ra." |
| 6 | Lợn Ủn ở chợ | Bán 1 giun | "Giun đất 5 thóc, giun vàng 150 thóc. Giun cầu vồng thì… cháu đào được, ta mua cả họ." |
| 7 | Gà Bán Thóc ở quầy thức ăn | Mua và ăn 1 món, hoặc ăn sống 1 giun | "Ăn đi cho có sức. Cám thì rẻ, suất thì no, ăn sống thì… cũng được, gà mà!" |
| 8 | Cụ Cục (qua thư Chim Chíp, không bắt quay về sảnh) | Nhận thưởng | "Giỏi! Thưởng 50 thóc, 3 cám và một quả trứng. Đem ấp ở chỗ Vịt Bông nhé — nở ra con gì là… tuỳ duyên!" |

- Mỗi bước có mũi tên chỉ đường sẵn có (`map.setDestination`) tới NPC kế tiếp. NPC đó có dấu **"!"** vàng trên đầu tới khi xong bước.
- Bỏ qua được: nút "Tôi biết chơi rồi" trong hội thoại đầu. Bỏ qua vẫn nhận 100 thóc khởi nghiệp, không nhận thưởng hướng dẫn.
- Người chơi cũ chưa xong hướng dẫn: giữ tiến độ cũ, ánh xạ 5 bước cũ sang bước tương ứng.

### 6.2. Thay đổi mã

- `account.go`: `TutorialSteps` mới (8 bước, ID ổn định), `ClaimTutorial` trả thóc + vật phẩm.
- `npc.go`: sự kiện `tutorial.next` đặt dấu "!" trên đúng NPC; lời chào có nhánh theo bước hướng dẫn.
- `panels.js` `dialogElder`: hội thoại nhiều trang có nút Tiếp/Bỏ qua; ảnh minh hoạ nhỏ (thanh độ no, nút chạy, nút giữ kéo).

### 6.3. Nghiệm thu

E2E người mới: vào sảnh nhận `say` từ Cụ Cục có tên mình trong ≤ 3 giây → hoàn thành 8 bước theo luồng → nhận đúng một lần thưởng. Người mới không bị trêu trong 5 phút đầu.

## 7. Xe buýt làng chạy liên tục

### 7.1. Tuyến và bến

- **Đường vành đai mới** (mục 5.1) nối cổng 9 khu của Trại Hypex theo vòng tròn quanh quảng trường. Dựng cả phía client (đường đất có cột đèn) lẫn máy chủ (đồ thị đường).
- **Tuyến vòng làng:** chạy theo chiều kim đồng hồ trên vành đai. Mỗi cổng khu có một **bến**: mái che nhỏ, ghế, biển "🚌 Bến {tên khu}", bảng giờ xe tới. Bến cửa sảnh là bến đầu tuyến.
- Thông số: 2 xe (3 xe nếu > 30 người ở ngoài trời), 12 ghế/xe, 9 m/giây, dừng 8 giây mỗi bến. Vòng dài ~550 m nên mỗi vòng ~2 phút 15 giây; mỗi bến có xe khoảng 70 giây một chuyến.
- Tài xế NPC "Bác Tài Gáy" và vài khách NPC ngồi sẵn cho có không khí (không chiếm ghế người chơi).
- **Vị trí xe tính theo đồng hồ máy chủ** (lịch tuần hoàn), nên xe chạy liên tục kể cả khi không ai ngồi. Bến biết chính xác lúc xe tới để hiện "Xe tới sau 0:42".

### 7.2. Lượt miễn phí và giá vé

| Thông số | Cũ | Mới |
| --- | --- | --- |
| Lượt miễn phí | 3 lượt/ngày, reset 0:00 | **2 lượt tối đa, hồi 1 lượt mỗi 5 phút** (tính theo giờ thực, hồi cả khi offline) |
| Hết lượt, tuyến vòng làng | — | **3 thóc**/chuyến |
| Hết lượt, xe liên trại Hypex ↔ Ven Suối | 10 thóc | **10 thóc** |
| Từ Ven Suối về Hypex | miễn phí | giữ miễn phí (không bị kẹt ở vùng khác) |
| Thú cưng "Vịt Mặc Cả" | — | giảm 20/35/50% giá thóc, làm tròn xuống |

- Lượt dùng chung cho xe vòng làng và xe liên trại.
- HUD: biểu tượng 🎫 cạnh ví thóc, ví dụ "🎫 1/2 · lượt sau 3:12".
- Hết lượt và không đủ thóc: không lên được xe, bến hiện "Đợi 3:12 để có lượt miễn phí, hoặc đi bộ — 140 m".

### 7.3. Lên và xuống xe

- Đứng trong bán kính 3 m quanh bến khi xe đang dừng, bấm E **"Lên xe"**. Chọn bến xuống trong danh sách có thời gian đi dự kiến, hoặc "Đi vòng, tự xuống".
- Trừ vé lúc lên (lượt hoặc thóc, có request ID). Xe tới bến đã chọn thì tự cho xuống; bấm **"Xuống ở bến kế"** thì xuống ở bến tiếp theo. Không xuống giữa đường.
- Đang ngồi xe: camera theo xe; người chơi hiện trong ô cửa sổ; không đào, không chạy, không giao dịch.
- Không lên được khi đang lái xe riêng, đang PK, đang chở bạn hoặc đang được chở.
- Mất kết nối khi đang ngồi: giữ ghế trong thời gian nối lại. Hết giờ thì người chơi được đặt ở bến gần nhất. Máy chủ khởi động lại giữa chuyến: vào lại ở bến gần vị trí cuối cùng đã lưu.
- **Xe liên trại:** tách khỏi tuyến vòng, đậu ở Bến xe Chim Chíp, xuất bến theo lịch mỗi 3 phút (`demo`: 90 giây), 8 ghế, mở cửa 20 giây trước giờ chạy. Giữ cơ chế vé "held → departed" và đối soát đang có trong `v11_room.go`/`v11.go`. Chỉ đổi từ "đặt vé rồi chờ đủ khách" sang lịch cố định.

### 7.4. Thay đổi mã

- Máy chủ:
  - `roads.go`: đồ thị đường, danh sách bến, lịch xe.
  - `npc.go`: actor `bus`, lên/xuống xe tổng quát hoá từ `syncRides` (xe riêng của người chơi, xe buýt và xe ôm đều là "phương tiện chở người").
  - `v11.go`: thay `TripsUsed/TravelDay` bằng `BusTokens`, `BusTokenAt`; đổi `Book`/`CancelBooking` sang tính lượt mới.
  - `actions.go`: hành động `bus_board`, `bus_alight`, `bus_stops`.
- Client: biển bến và bảng giờ, thẻ chọn bến xuống, chỉ báo 🎫 trên HUD, dựng xe chạy trên vành đai.
- Test cũ phải sửa: test đặt vé và xe liên trại (`FreeTrips`, `BusFare`), test xe chở bạn qua bến.

### 7.5. Nghiệm thu

- Test lịch: vị trí xe theo thời gian khớp vòng; ETA đúng ±1 giây.
- Test lượt: 2 lượt → dùng hết → trả 3 thóc → 5 phút sau có lại 1 lượt; tối đa 2.
- E2E: lên ở bến cửa sảnh, xuống ở bến Vườn giun, vị trí cuối nằm trong bán kính bến; xe đầy thì từ chối người thứ 13; về từ Ven Suối luôn miễn phí.

## 8. Xe ôm

### 8.1. Số lượng vừa đủ

- Số xe ôm chạy rong trong một phòng = `clamp(1 + ceil(người_ngoài_trời / 2), 2, 6)`: 1–2 người có 2 xe, 3–4 người có 3 xe… tối đa 6. Thêm **1 xe đậu chờ cố định** ở "Bến xe ôm cây đa" cạnh quảng trường.
- Xe rong chia đều trên các đoạn đường: khi sinh xe mới, chọn đoạn xa các xe ôm khác nhất. Một người đi bộ trên đường lớn thường gặp xe ôm sau khoảng 30–60 giây.
- Nhân vật: "Gà Xe Ôm" đội mũ bảo hiểm, áo khoác xanh, xe máy cũ. Tên ngẫu nhiên vui như "Anh Tư Bốc Đầu", "Chú Sáu Tay Lái Lụa", "Cô Ba Xe Ôm Công Nghệ".

### 8.2. Vòng trạng thái

`chạy rong → tiến lại gần → mời → chờ trả lời (15 giây) → chở → trả khách → nghỉ 20 giây → chạy rong`

- **Điều kiện mời:**
  - người chơi đang đi bộ ở khu ngoài trời, không ngồi xe, không đào, không PK, không giao dịch;
  - có ít nhất 1 thóc;
  - chưa được mời trong 3 phút;
  - chưa từ chối 2 lần trong 10 phút (nếu đã từ chối thì im 15 phút);
  - không phải người mới trong 2 phút đầu.
- **Tiến lại gần:** xe giảm tốc, tạt vào cách người chơi 2,5 m, bóp còi nhẹ, nói câu mời. Ví dụ:
  - "Ê, đi đâu đó? Lên xe anh chở, 1 thóc thôi, nhanh gấp đôi chạy bộ!"
  - "Trời nắng vậy đi bộ chi cho mệt, lên đây anh chở, máy lạnh tự nhiên!"
- **Người chơi trả lời:** thẻ nổi "🛵 {tên tài xế} mời đi xe ôm · [Đi] [Không, cảm ơn]", phím E là Đi, X là Không.
  - Bấm **Đi**: bản đồ mở ở chế độ "Chọn điểm đến", chạm vào một khu sẽ hiện giá (1 hoặc 2 thóc) và thời gian đi; xác nhận để lên xe.
  - Bấm **Không**: tài xế đáp một câu vui rồi chạy tiếp. Ví dụ: "Thôi được, đi bộ cho khoẻ, gặp lại ở ngã ba nha!" hoặc "Không đi thì thôi, anh đi kiếm khách khác, khỏi năn nỉ!"
- **Gọi chủ động:** đứng ở bến xe ôm cây đa, hoặc bấm nút "🛵 Vẫy xe ôm" khi có xe ôm trong 30 m (xe đó sẽ quay lại đón).

### 8.3. Giá và đường đi

- Giá theo quãng đường đi theo đường làng, từ điểm đón tới cổng khu đích: **≤ 90 m là 1 thóc, > 90 m là 2 thóc**. "Vịt Mặc Cả" giảm giá nhưng không xuống dưới 1 thóc.
- Trừ thóc lúc lên xe (request ID, ghi sổ cái lý do `xeom_fare`). Nếu máy chủ huỷ chuyến (lỗi, xe kẹt) thì hoàn đủ.
- Tốc độ 8,5 m/giây, đi đường ngắn nhất trên đồ thị đường. Không vào khu cấm lái: trả khách ở cổng. Khu cho lái thì trả gần điểm đến.
- Đang chở: người chơi ngồi sau tài xế (ghế `rideOf`). Tài xế thỉnh thoảng nói chuyện ("Bám chắc nha, qua ổ gà đó!", "Hồi xưa anh cũng đào giun, giờ chạy xe ôm thu nhập ổn định hơn!"). Nút "Xuống đây" cho xuống ở điểm an toàn kế tiếp, không hoàn tiền.
- Trả khách: "Tới rồi! Nhớ đánh giá 5 sao… à ở đây chưa có sao, cảm ơn là được!"
- Người chơi mất kết nối khi đang ngồi: xe dừng chờ hết thời gian nối lại, rồi đặt người chơi ở lề đường; tiền đã trả không hoàn.

### 8.4. Nghiệm thu

- Test số lượng xe ôm theo số người.
- Test điều kiện mời và các hạn mức (3 phút, từ chối 2 lần).
- Test giá 1/2 thóc theo quãng đường, và mức sàn 1 thóc khi có "Vịt Mặc Cả".
- Test hoàn tiền khi máy chủ huỷ chuyến.
- E2E: đi bộ tới gần, nhận `offer`, đồng ý chọn Chợ thóc, bị trừ đúng thóc, tới cổng chợ; từ chối thì nhận `say` và không bị mời lại trong 3 phút.

## 9. Làng sống: NPC tự tương tác

### 9.1. Giao thông NPC

- Mỗi phòng ở Hypex có 8 actor giao thông:
  - 3 xe đạp: "Gà Đạp Xe Đi Chợ", có giỏ rau;
  - 2 xe máy: "Gà Shipper", có thùng hàng;
  - 2 ván trượt: "Gà Teen", đội mũ ngược;
  - 1 nhóm đi bộ: "Cụ Gà Tập Dưỡng Sinh" vung tay.
- Chạy trên vành đai và đường nhánh, đi bên phải đường. Gặp người chơi chắn đường trong 3 m thì giảm tốc, bóp còi "Bíp bíp!", có lúc nói "Tránh đường nha!". Không bao giờ đẩy người chơi (không va chạm vật lý).
- Thỉnh thoảng (theo hạn mức 5.2) chào người chơi đi ngang: "Chào buổi sáng!", "Hôm nay giun béo không?".
- Ngày/đêm (mục 16): ban đêm bớt xe, xe bật đèn.

### 9.2. Cameo nhắm vào người chơi

Đạo diễn sinh một kịch bản ngắn có đường đi tính quanh vị trí người chơi, gửi `scene`/`say` cho mọi người trong 35 m. Chỉ người được nhắm tới nhận viền cam và dòng nhật ký.

| Cameo | Điều kiện | Diễn biến |
| --- | --- | --- |
| **Thiếu gia ô tô** ("Gà Thiếu Gia", ô tô đỏ, kính râm) | Đang đi bộ trên đường, không ngồi xe; cho phép trêu; cấp ≥ 2 | Xe tới từ phía sau 40 m, chậm lại sát người chơi 1,5 giây: "Ơ kìa, thời buổi này còn có gà đi bộ à?" → bóp còi, rồ ga phóng đi: "Hahaha! Bíp bíp! Mua xe đi cưng!" Nếu người chơi **có xe đạp**: "Xe đạp à? Dễ thương ghê, như đồ chơi!" Nếu **có ô tô**: "Ồ, xe xịn đấy! Hôm nào đua một vòng không?" (chuyển sang nể) |
| **Hai mẹ con gà** | Người chơi đang đói, vừa thua PK, hoặc đứng yên > 60 giây ngoài đường | Đi ngang qua. Mẹ: "Con thấy chưa, không chịu học thì sau này lại như {anh/chị/bạn} kia đấy." Con (ngẫu nhiên): "Nhưng {anh/chị/bạn} ấy dễ thương mà mẹ…" / "Con sẽ học đào giun thật giỏi!" / "Mẹ ơi, {anh/chị/bạn} ấy đang nhìn mình kìa!" |
| **Gà Shipper hỏi đường** | Đi bộ ở quảng trường | "Ơ cho hỏi Chợ thóc đi lối nào?" Người chơi chọn 1 trong 3 hướng: đúng thì +1 thóc + câu cảm ơn; sai thì shipper đi lạc, lát sau quay lại: "Chỉ đường kiểu gì vậy trời!" |
| **Gà Nhà Báo** | Vừa đào được giun từ Hiếm trở lên | Chạy tới cầm micro: "Cảm nghĩ khi đào được {giun}?" Chọn 1 trong 3 câu trả lời vui, mở thẻ khoe giun có sẵn |
| **Gà Bán Rong** | Độ no < 30, ở xa chợ > 60 m | "Cám nóng đây! Đắt hơn ngoài chợ tí xíu nhưng giao tận mỏ!" Bán cám/rau giá +20% |
| **Cụ Gà Dưỡng Sinh** | Vừa chạy hụt hơi 3 lần trong 2 phút | "Chạy gì mà như ma đuổi! Hít vào… thở ra… Từ từ thôi con." |

### 9.3. Trận biểu diễn ở Sân tỷ thí (và ở khu lễ hội khi có ngày hội)

- Cứ 4 phút một trận, mỗi trận 60–80 giây, khi sân không có trận PK thật. Có người chơi bắt đầu PK thật thì trận biểu diễn dừng ngay (người thật được ưu tiên).
- Diễn viên: 2 đấu sĩ NPC (ví dụ "Gà Tía Mặt Sắt" và "Gà Ô Lì Đòn"), trọng tài Chó Mực, 6–10 khán giả (ambient residents đã có).
- Kịch bản (máy chủ gửi một gói `scene {id, kind:"arena_show", seed, at}`; mọi client dựng cùng một dòng thời gian từ seed nên ai cũng thấy như nhau mà tốn rất ít băng thông):
  1. Chó Mực: "Hai bên chào nhau! Không mổ mắt, không đá dưới cánh! 3… 2… 1… Bắt đầu!"
  2. Hiệp mổ: hai gà lao vào nhau, mổ, nhảy lùi; tỉ số hiện trên bảng gỗ ở sân.
  3. Khán giả, mỗi người một câu theo diễn biến:
     - cổ vũ: "Cố lên Gà Tía!";
     - bàn tán: "Tôi cược 5 thóc con lông ô thắng!";
     - cười nhạo: "Mổ đi chứ, gãi ngứa à!";
     - khi có cú ngã: "Ối giời, sao vàng bay quanh đầu kìa!".
  4. Kết thúc: đấu sĩ thắng dang cánh; đấu sĩ thua nằm nghiêng có sao vàng 2 giây rồi đứng dậy. Chó Mực: "KO! Gà Tía thắng! Cả hai về ăn cám rồi tập tiếp nhé." Một khán giả: "Thôi về, mai xem tiếp!"
- Người chơi đứng xem được mời: "Muốn lên sàn thử không? Gặp Chó Mực nhé!" Đây là cầu nối sang PK thật. Không có cược bằng thóc thật trong bản này: tránh cờ bạc và lạm phát.
- Ngày hội cộng đồng (`setFestival` đang có): trận biểu diễn chuyển sang sân khấu ở Ao và vườn, có thêm pháo giấy và đèn lồng.

### 9.4. Chủ quán chủ động

| NPC | Khi nào | Nói và làm gì |
| --- | --- | --- |
| Dê Thợ Máy (Cửa hàng xe) | Lần đầu bước vào tiệm | "Ồ khách mới! Hôm nay giảm **10%** cho xe đầu tiên, hạn 24 giờ. Đi bộ hoài teo cả đùi gà đó!" Máy chủ cấp phiếu `phieu_giam_xe_10` (một lần mỗi tài khoản); `BuyVehicle` tự áp phiếu |
| Dê Thợ Máy | Đủ thóc mua xe đạp mà chưa có xe | "Đủ tiền mua xe đạp rồi đó, đạp cho thon đùi!" |
| Lợn Ủn (chợ) | Người chơi có giun từ Hiếm trở lên trong túi | "Ui chà, giun phát sáng! Bán cho tôi đi, hôm nay giá đẹp!" Mỗi ngày Lợn Ủn chọn một loài "thu mua giá cao +10%", hiện trên bảng phấn |
| Vịt Bông (trạm ấp) | Có trứng mà ổ ấp trống | "Trứng để trong túi lâu thành trứng luộc đó! Đem đây cô ấp cho." |
| Bò Mộc (xóm) | Đủ cấp và đủ thóc mua đất | "Muốn có nhà riêng không? Gặp ông Chủ Đầu Tư bên kia, rồi về đây tôi xây cho!" |
| Chim Chíp (bến) | Người chơi đứng ở bến | "Xe tới sau {eta} giây! Còn {n} lượt miễn phí đó." |
| Cụ Cục | Người chơi lên cấp, hoặc quay lại sau ≥ 1 ngày | "Lên cấp {cap} rồi à? Giỏi! Nhớ ghé nhận quà." / "Về rồi đấy à {ten}! Làng nhớ cháu lắm… nhất là Lợn Ủn, nó nhớ giun của cháu." |

### 9.5. NPC phản ứng theo trạng thái người chơi

- **Đói (< 20):** NPC gần nhất nhắc, ví dụ "Bụng kêu to thế, ra chợ ăn gì đi!" hoặc "Không có thóc thì ăn sống một con giun đi, gà mà!"
- **Giun thoát** (gần Lợn Ủn hoặc người đào cạnh bên): "Nó chuồn rồi! Lần sau đừng giật mạnh thế."
- **Thua PK:** khán giả an ủi hoặc trêu nhẹ: "Thua keo này ta bày keo khác!"
- **Lên cấp:** vài NPC quanh đó vỗ cánh chúc mừng (hoạt ảnh `greet` sẵn có).
- **Mặc đồ mới:** chủ tiệm khen. Đeo vương miện thì: "Ồ, vua giun về làng!"

## 10. Cây và vật che tầm nhìn

**Hiện trạng:** cây ngoài làng được gộp chung vào một mesh lớn mỗi ô 60 m để giảm lượt vẽ. Vì vậy không mờ riêng từng cây được, và đi sau cây là mất dấu gà.

**Cách làm:**

1. Cây không đi vào bước gộp vật liệu nữa. Mỗi ô 60 m có một `InstancedMesh` cho thân và một cho tán (mỗi biến thể màu một mesh), kèm danh sách cây (vị trí, bán kính tán, chiều cao) trong lưới tra cứu 10 m.
2. Mỗi khung (hoặc 10 lần/giây): lấy đoạn camera → ngực gà và camera → điểm nhìn. Tra các ô lưới mà đoạn đi qua. Cây nào có tán hoặc thân cắt đoạn (khoảng cách < bán kính + 0,3 m) và nằm giữa hai đầu đoạn (5–95% chiều dài) thì bị đánh dấu.
3. Cây bị đánh dấu: thu nhỏ bản thể trong mesh đục về 0, rồi vẽ cây đó trong một `InstancedMesh` "bóng mờ" dùng chung (trong suốt 25%, không ghi độ sâu). Tối đa 16 cây mờ cùng lúc; hiện lại dần trong 0,25 giây.
4. Mở rộng cùng cơ chế cho đống rơm, chuồng ở Xóm, quầy ở chợ. Kiến trúc lớn (vách, mái, cổng) đã có cơ chế mờ riêng (`occluder`, `roofBlocks`).
5. Không kéo camera xuyên qua cây (cảm giác giật). Chỉ làm mờ.

**Nghiệm thu:** ảnh chụp đi sau hàng cây ở làng, sau chuồng ở Xóm, sau quầy ở chợ: luôn thấy gà; số lượt vẽ không tăng quá 5%.

## 11. Kho đồ (vật phẩm)

Hiện chỉ có `Worms`, `Food`, `Eggs` (số nguyên). Thêm kho vật phẩm chung làm nền cho thưởng cấp, xây nhà, hạt giống, trứng, phiếu giảm giá.

- `Account.Items map[string]int`; bảng `ItemDefs` gồm id, tên, emoji, nhóm, giá mua, giá bán lại, có giao dịch được không, giới hạn chồng.
- Kho tối đa **200 đơn vị** vật liệu và hạt giống, tách khỏi túi giun 60.
- Chuyển dữ liệu: `Eggs` → `Items["trung_thuong"]`; `Food` giữ map riêng ở bản đầu, giao diện gộp chung tab "Kho".

| Nhóm | Vật phẩm | Mua ở đâu | Giá mua | Giao dịch |
| --- | --- | --- | --- | --- |
| Vật liệu | Rơm, Gỗ, Đinh, Đá, Gạch, Ngói | Tiệm vật liệu của Bò Mộc (Xóm) | 3 / 8 / 2 / 6 / 12 / 10 thóc | được (từ cấp 3) |
| Hạt giống | Cải, Cà rốt, Bí ngô, Hoa | Quầy thức ăn (chợ) | 3 / 5 / 12 / 6 thóc | được |
| Nông sản | Cải, Cà rốt, Bí ngô, Bó hoa | thu hoạch | bán ở quầy thức ăn | được |
| Trứng | Trứng thường, Trứng đốm, Trứng vàng | thưởng cấp, nhiệm vụ, sự kiện | không bán | không |
| Phiếu | Giảm 10% xe đầu tiên | Dê Thợ Máy tặng | — | không |

- **Nguồn vật liệu khác:** đào ở ụ "Gốc cây" có 8% rơi thêm 1 Gỗ, ụ "Đất ẩm" có 6% rơi thêm 1 Đá. Phần rơi thêm không ảnh hưởng bảng giun. Bảng việc làng và thưởng cấp cũng cho vật liệu.

## 12. Lên cấp tặng thưởng

- Đào cho XP (mục 4.4), ăn vẫn là nguồn XP chính. Cấp tối đa giữ 10 ở bản này; mở tới 20 là một giai đoạn sau (cần thêm đường cong XP và nội dung).
- Thưởng nhận **tự động khi lên cấp** (idempotent theo `LevelRewards[cap]`), hiện bảng quà và Cụ Cục chúc mừng. Mốc trứng cũ (cấp 5/7/9) gộp vào bảng; ai đã nhận thì không nhận lại.

| Cấp | Thóc | Vật phẩm | Mở khoá |
| --- | --- | --- | --- |
| 2 | 50 | 3 cám, 2 hạt cải | — |
| 3 | 80 | 10 rơm, 5 gỗ, 2 hạt cà rốt | mua đất, giao dịch |
| 4 | 100 | 1 trứng thường, 2 rau trộn | — |
| 5 | 120 | 10 gỗ, 5 đá, 1 trứng thường | xe máy |
| 6 | 150 | 2 hạt bí ngô, 10 đinh | lô rộng |
| 7 | 180 | 10 gạch, 1 trứng thường | ô tô, nghề thợ xây |
| 8 | 200 | 1 trứng đốm, 2 suất ăn | — |
| 9 | 250 | 10 ngói, 10 gạch, 1 trứng thường | — |
| 10 | 300 | 1 trứng vàng, danh hiệu "Gà Lão Làng" | lập Hội Gà |

## 13. Mua đất và xây nhà

### 13.1. Hiện trạng

Mỗi người có nhà riêng là một phòng riêng ("Tổ nhỏ" miễn phí). Nâng hạng đất trong menu nhà: Mảnh đất riêng 1.200 thóc, Đất mở rộng 3.000 thóc, từ cấp 3. Bò Mộc xây chuồng 500 thóc theo màu mái. Có hợp đồng thợ xây giữa người chơi.

### 13.2. Chủ đầu tư và lô đất

- NPC mới **"Ông Chủ Đầu Tư"** (gà trống bụng phệ, vest, cầm bản đồ cuộn) ở **"Văn phòng đất nền"**: một kiosk mái rơm cạnh cổng Xóm Rơm Vàng.
  - Lời chào: "Đất ven ao, view triệu đô, sổ đỏ… à sổ rơm chính chủ! Mua ngay kẻo hết lô đẹp!"
  - Khi người chơi chưa đủ thóc: "Chưa đủ thóc à? Không sao, đào thêm vài con giun vàng là có ngay!"
- Mua đất chuyển từ menu nhà sang hội thoại với Chủ đầu tư. Giao diện là bản đồ khu đất có các lô vẽ sẵn.

| Lô | Kích thước khu nhà | Giá | Cấp |
| --- | --- | --- | --- |
| Tổ nhỏ (sẵn có) | 9 × 8 | 0 | 1 |
| Lô tiêu chuẩn | 13 × 11 | 1.200 | 3 |
| Lô rộng | 17 × 14 | 3.000 | 6 |

| Vị trí (phụ phí, chỉ đổi cảnh quanh nhà) | Phụ phí |
| --- | --- |
| Cạnh đường làng | +0 |
| Ven ao (có ao nhỏ, lau sậy) | +300 |
| Đồi thông (cây thông, nhìn ra núi) | +600 |

- **Nhà vẫn là phòng riêng**, nên không có chuyện hết lô và mua bán được không giới hạn người. Để làng có cảm giác "khu nhà thật", Xóm Rơm Vàng có **8 lô trưng bày**: mỗi kênh hiển thị nhà của người chơi đang online trong kênh đó, ưu tiên bạn bè. Lô có biển tên chủ và cổng vào thăm. Khi không đủ người thì lô hiện nhà mẫu của Chủ đầu tư ("NHÀ MẪU – LIÊN HỆ CHỦ ĐẦU TƯ"). Việc này ở giai đoạn sau của mục này (P5b).

### 13.3. Chọn thiết kế và vật liệu, Bò Mộc xây

| Thiết kế | Rơm | Gỗ | Gạch |
| --- | --- | --- | --- |
| **Chuồng nhỏ** (lô tiêu chuẩn trở lên) | 200 thóc + 20 rơm + 5 gỗ · 10 phút | 400 thóc + 20 gỗ + 10 đinh · 20 phút | 800 thóc + 20 gạch + 10 gỗ + 6 ngói · 40 phút |
| **Nhà hai gian** (lô rộng) | — | 900 thóc + 40 gỗ + 20 đinh + 10 rơm · 40 phút | 1.800 thóc + 40 gạch + 20 gỗ + 12 ngói · 60 phút |

- Vật liệu ảnh hưởng ngoại hình (mái rơm, vách gỗ hay tường gạch) và **độ ấm cúng**. Ấm cúng càng cao thì ngủ ổ ở nhà hồi độ no càng nhanh: rơm +1/phút, gỗ +1,5/phút, gạch +2/phút. Sức chứa khách: rơm 8, gỗ 12, gạch 16.
- Bò Mộc xây theo giờ, chạy cả khi offline. Thuê thợ là người chơi (hợp đồng đang có) để rút ngắn thời gian: hợp đồng thêm điều kiện "chủ đã gom đủ vật liệu".
- Nâng cấp: đổi rơm lên gỗ, gỗ lên gạch, chỉ trả phần chênh lệch vật liệu và một nửa thóc. Xây xong có tiệc tân gia (tính năng tiệc đang có).
- Chuyển dữ liệu: chuồng đã xây trước đây coi như "Chuồng nhỏ – Gỗ".

### 13.4. Nghiệm thu

- Test mua lô (đủ, thiếu cấp, thiếu thóc), phụ phí vị trí, xây với thiếu vật liệu, chạy giờ khi offline, nâng cấp vật liệu, hợp đồng thợ có vật liệu.
- E2E: mua lô tiêu chuẩn ven ao, xây chuồng rơm, chờ, thấy nhà mới trong phòng nhà riêng.

## 14. Hạt giống và trồng cây

| Hạt | Thời gian lớn | Thu hoạch | Công dụng |
| --- | --- | --- | --- |
| Cải | 10 phút | 3 cải | ăn +15 độ no, bán 4 thóc |
| Cà rốt | 20 phút | 3 cà rốt | ăn +25, bán 7 |
| Bí ngô | 45 phút | 1 bí ngô | ăn +80, bán 25 |
| Hoa | 30 phút | 1 bó hoa | tặng bạn (+thân thiết với bạn đồng hành của họ), trang trí nhà |

- **Trồng ở đâu:**
  - luống rau ở nhà riêng: đồ trang trí "Luống rau" trở thành ô trồng được, mỗi luống 1 cây;
  - **12 luống cộng đồng** ở Ao và vườn: mỗi người mượn tối đa 1 luống, giữ 2 giờ, cây thuộc về người trồng, người khác không hái được.
- **Tưới:** mỗi lần trồng được tưới 1 lần, rút 25% thời gian còn lại. Có thể tưới giúp luống của bạn bè (+1 thân thiết, không được thu hoạch).
- **Héo:** để quá 3 lần thời gian lớn mà không thu thì cây héo, chỉ còn 1 "phân ủ" (bón cho lần trồng sau: +1 nông sản).
- Thời gian lớn chạy theo giờ thực (cả khi offline), giống ấp trứng.
- Nghiệm thu: test vòng trồng → tưới → chín → thu → héo; không hái được luống người khác; một người chỉ mượn 1 luống cộng đồng.

## 15. Trứng và bạn đồng hành có kỹ năng

### 15.1. Trứng và loài

| Trứng | Nguồn | Loài có thể nở (tỉ lệ) |
| --- | --- | --- |
| Thường | thưởng hướng dẫn, cấp 4/5/7/9, bảng việc | Gà con 60% · Vịt con 25% · Ngỗng con 15% |
| Đốm | cấp 8, sự kiện, điểm danh ngày 7 | Chim cút 35% · Gà tre 35% · Gà ác 30% |
| Vàng | cấp 10, sự kiện hiếm | Công con 50% · loài của trứng thường hoặc đốm (ngẫu nhiên) 50% |

Ấp ở Trạm ấp như hiện tại; bộ `live` ấp 30 phút. Mỗi người ấp 1 trứng một lúc. Vẫn tối đa 3 bạn đồng hành mang theo người; có thêm chuồng nuôi ở nhà riêng chứa 6 con (cần đã xây chuồng).

### 15.2. Kỹ năng

Chỉ **bạn đồng hành đang đi theo** mới có tác dụng. Mức kỹ năng theo độ thân thiết đang có: Lv1 (0–49), Lv2 (50–149), Lv3 (150+).

| Loài | Kỹ năng | Lv1 / Lv2 / Lv3 |
| --- | --- | --- |
| Gà con "Mỏ Thính" | **Đánh hơi giun:** tăng tỉ lệ các bậc từ Khá hiếm trở lên (nhân, rồi chuẩn hoá lại tổng); ụ có giun ngon nhất lấp lánh | +4% / +7% / +10% |
| Vịt con "Mặc Cả" | **Giảm giá vé** xe buýt và xe liên trại (xe ôm tối thiểu vẫn 1 thóc) | −20% / −35% / −50% |
| Ngỗng con "Lưng Chắc" | **Cõng chạy:** thêm thể lực chạy; Lv3 mở "Cưỡi ngỗng" chạy ×1,5 trong 20 giây, hồi 3 phút | +2 / +3 / +4 giây |
| Chim cút "Chân Nhanh" | Tăng tốc đi bộ | +5% / +8% / +10% |
| Gà tre "Bụng Khoẻ" | Giảm tốc tụt độ no | −10% / −18% / −25% |
| Gà ác "Tay Buôn" | Bán giun cho Lợn Ủn giá cao hơn | +5% / +8% / +12% |
| Công con "May Mắn Kép" | Cơ hội đào được 2 con cùng loài | 3% / 5% / 8% |

- **Giới hạn cộng dồn:** tổng tăng may mắn khi đào ≤ +10% (sau này có bùa may mắn cũng không vượt); giảm giá vé ≤ 50%; tốc độ đi bộ cộng mọi nguồn ≤ +10%.
- Kỹ năng phải hiện rõ: thẻ bạn đồng hành ghi mô tả và mức; khi có tác dụng thì hiện chữ nhỏ ("🦆 Mặc cả −35%" trên thẻ giá vé, "🐤 Đánh hơi!" khi ụ lấp lánh).
- Gà con đã nở trước đây: loài giữ "Gà con", được kỹ năng "Mỏ Thính" ngay.

### 15.3. Nghiệm thu

- Test phân phối loài theo loại trứng: mô phỏng 10.000 lần, lệch tối đa ±2%.
- Test mỗi kỹ năng chỉ có tác dụng khi đang đi theo, đúng mức theo thân thiết, đúng giới hạn cộng dồn.
- Kiểm tra tổng xác suất bảng giun luôn bằng 1 sau khi áp may mắn.

## 16. Logic bổ sung đề xuất

| Ưu tiên | Tính năng | Tóm tắt |
| --- | --- | --- |
| Cao | **Điểm danh 7 ngày** | Ngày 1–6: 20/30/40/50/60/80 thóc kèm cám hoặc hạt giống; ngày 7: 1 trứng đốm. Lỡ một ngày thì quay về ngày 1. Cụ Cục trao quà bằng bong bóng khi vào game |
| Cao | **Nghỉ ổ** | Ngồi ổ ở sảnh: độ no tụt chậm một nửa (đã ở mục 2). Ngồi ổ cạnh nhau với bạn bè: +1 thân thiết/phút cho bạn đồng hành của cả hai |
| Vừa | **Ngày/đêm** | Một ngày game = 48 phút thực (24 phút ngày, 24 phút đêm). Đêm: đèn lồng sáng, NPC giao thông về nhà, Giun phát sáng ×1,5 tỉ lệ (vẫn trong giới hạn tổng). NPC chào theo buổi |
| Vừa | **Thời tiết** | Mưa nhẹ ngẫu nhiên khoảng 10% thời gian: giun ngoi lên nên hồi đất −30%; xe ôm "tăng giá mưa" +1 thóc (NPC giải thích hài: "Mưa to, chở gà ướt, phụ thu khăn lau!") |
| Vừa | **Nhiệm vụ của NPC** | Lợn Ủn cần 5 giun béo, Bò Mộc cần 10 rơm, Vịt Bông cần 1 bó hoa…; thưởng vật liệu hoặc trứng. Mở rộng "bảng việc làng" đang có |
| Vừa | **Bảng xếp hạng tuần** | Giun dài nhất, nhiều giun hiếm nhất, thắng PK nhiều nhất; thưởng danh hiệu, không thưởng thóc lớn |
| Thấp | **Câu cá ở ao** | Đặc tả riêng sau: điểm câu ở cầu ao, mini-game phao, cá bán hoặc ăn; dùng chung khung giằng co của đào giun |
| Thấp | **Mở cấp 11–20** | Đường cong XP mới, nội dung mở khoá (lô đồi, nhà gạch hai tầng, trứng hiếm) |

## 17. Bảng cân bằng tổng hợp (bộ `live`)

| Hạng mục | Giá trị |
| --- | --- |
| Độ no tối đa / tụt khi online / khi ngồi ổ | 100 / 4,5 mỗi phút / 2,25 mỗi phút |
| Trừ mỗi lượt đào | 0,4 |
| Đi chậm / kiệt sức sau khi ở 0 | 3 phút / 6 phút |
| Quà khởi nghiệp / thưởng hướng dẫn | 100 thóc / 50 thóc + 3 cám + 1 trứng thường |
| Ăn sống giun | 60% độ no, 0 XP, 0 phí |
| Đi bộ / chạy / thể lực / khoá khi hụt hơi | 4,6 m/s / ×1,65 / 6 giây / 5 giây |
| Hồi đất / lượt đào có thưởng mỗi ngày | 8 giây / 120 |
| XP khi đào (theo độ hiếm) | 2 / 4 / 8 / 15 / 25 |
| Lượt xe miễn phí | tối đa 2, hồi 1 lượt mỗi 5 phút |
| Vé vòng làng / liên trại / về Hypex | 3 / 10 / 0 thóc |
| Xe buýt vòng làng | 2–3 xe, 12 ghế, 9 m/s, dừng 8 giây |
| Xe liên trại | mỗi 3 phút, 8 ghế |
| Xe ôm | 2–6 xe + 1 đậu chờ; 8,5 m/s; 1 thóc (≤ 90 m) / 2 thóc |
| Ấp trứng / Bò Mộc xây | 30 phút / theo bảng mục 13.3 |

**Dòng thóc ước tính với người chơi trung bình 1 giờ** (để soi lạm phát):

- Vào:
  - đào khoảng 5 lượt/phút, nên chạm giới hạn 120 lượt có thưởng của ngày sau khoảng 25 phút đào liên tục; kỳ vọng ~9,6 thóc/lượt nếu bán hết, tức tối đa ~1.150 thóc/ngày;
  - thưởng cấp tới cấp 4 (~230);
  - nhiệm vụ ngày (~60);
  - quà khởi nghiệp và hướng dẫn (150, chỉ một lần).
- Ra:
  - ăn ~55;
  - vé xe 0–15;
  - xe ôm 2–6;
  - phí chế biến;
  - đồ thời trang, vật liệu, đất, nhà (khoản chi lớn, tuỳ người).

Số liệu thật lấy từ sổ cái sau 1 tuần chơi thử và chỉnh lại.

## 18. Giao thức mạng (thêm và đổi)

Client → máy chủ:

| Gói | Ý nghĩa |
| --- | --- |
| `in {x, z, run}` | thêm cờ chạy |
| `act dig_start {id}` | trả thêm `worm, rarity, seed, params, peckMs` |
| `act dig_finish {id, events}` | thay `dig_stop` |
| `act eat_raw {worm}` | ăn sống |
| `act bus_board {stop, to}` · `bus_alight` · `bus_stops` | xe buýt |
| `act xeom_reply {offer, yes, to}` · `xeom_call` · `xeom_leave` | xe ôm |
| `act land_buy {lot, place}` · `house_build {design, material}` · `house_upgrade` | đất, nhà |
| `act item_buy {id, n}` · `item_sell {id, n}` | kho đồ |
| `act plant {bed, seed}` · `water {bed}` · `harvest {bed}` | trồng cây |
| `act pet_ride` | cưỡi ngỗng |
| `act prefs {npcChatter, npcTease}` | cài đặt NPC |
| `act npc_choice {scene, choice}` | trả lời cameo (chỉ đường, phỏng vấn) |

Máy chủ → client:

| Gói | Ý nghĩa |
| --- | --- |
| `snap` thêm `a: [...]` và cờ `r` cho người chơi | actor và dáng chạy |
| `vit` thêm `stam, lock` | thể lực |
| `say {id, name, text, ttl, to?, mood?}` | lời NPC |
| `offer {id, kind, driver, ttl}` | lời mời xe ôm |
| `scene {id, kind, seed, at, actors}` | cảnh dựng sẵn (trận biểu diễn, cameo) |
| `tokens {n, nextAt}` | lượt xe miễn phí |
| `reward {level, thoc, items}` | quà lên cấp |
| `mark {npc, kind:"quest"}` | dấu "!" hướng dẫn |

Mọi `act` mới đều có `rid` và được ghi nhớ kết quả như hiện tại (`Remembered`/`Remember`).

## 19. Dữ liệu và chuyển đổi

- **Account** thêm:
  - `StarterGranted`, `BusTokens`, `BusTokenAt`;
  - `Items`, `Prefs`;
  - `NPCSeen map[string]string` (NPC đã chào: ngày), `Coupons`, `CheckIn {Day, Streak}`, `Beds` (luống đang mượn).
- **Companion** thêm `Ability` (suy ra từ `Species`). **Home** thêm `Lot`, `Place`, `House {Design, Material, State, DoneAt}`.
- `normalize()` điền mặc định cho bản lưu cũ:
  - `TripsUsed` → `BusTokens = 2`;
  - `Eggs` → `Items["trung_thuong"]`;
  - `Coop` đã xây → `House{Design:"chuong_nho", Material:"go"}`;
  - mốc trứng đã nhận → đánh dấu trong `LevelRewards`.
- Thứ tự triển khai đảm bảo bản lưu mới vẫn đọc được bởi máy chủ cũ trong lúc chuyển đổi: trường mới đều `omitempty`, không đổi kiểu trường cũ.

## 20. Hiệu năng và mạng

- **Actor mỗi phòng Hypex:** 2–3 xe buýt, 3–7 xe ôm, 8 giao thông, tối đa 2 cameo, 1 cảnh sân, tức khoảng 20–25 actor. Máy chủ chỉ cập nhật vị trí dọc đường: chi phí rất nhỏ so với ngân sách tick 50 ms (tick p95 hiện 0,24 ms).
- **Băng thông:** chỉ gửi actor trong 70 m, khoảng 10 actor × ~24 byte × 10 lần/giây ≈ 2,4 KB/giây mỗi người. Cảnh sân và cameo gửi một gói seed duy nhất.
- **Client:**
  - mỗi actor là 1 mô hình đã "nướng" khớp như NPC cư dân (~6 lượt vẽ);
  - xe ở xa > 60 m dùng mô hình giản lược;
  - bong bóng là sprite canvas có cache theo câu.
- **Load test:** bot tải (`server/cmd/loadtest`) thêm hành vi chạy, lên xe buýt, gọi xe ôm, đào giằng co. Mục tiêu giữ tick p95 < 2 ms với 1.000 bot.

## 21. Kiểm thử

| Mức | Nội dung |
| --- | --- |
| Hàm thuần | độ no và các giai đoạn; thể lực; lượt xe và giá vé; mô phỏng giằng co (tỉ lệ bắt, máy chủ và client trùng kết quả); bảng giun sau may mắn (tổng = 1); thưởng cấp idempotent; trồng cây; đất và nhà; kỹ năng thú cưng và giới hạn cộng dồn; chọn câu thoại theo điều kiện và hạn mức |
| Máy chủ (phòng) | xe buýt theo lịch, lên/xuống, đầy chỗ; vòng trạng thái xe ôm, hoàn tiền; đạo diễn không vượt hạn mức trêu và cameo; tắt trêu thì không bị trêu |
| E2E WebSocket | người mới đi trọn hướng dẫn; 0 thóc 0 độ no vẫn thoát được; đi xe buýt 2 lượt miễn phí rồi trả thóc; xe ôm tới chợ; đào giun hiếm thoát và không tính hạn ngày; mua đất và xây nhà |
| Hồi quy | toàn bộ 40 test hiện có (sửa các test xe buýt theo luật mới) |
| Ảnh / tay | bong bóng NPC đọc rõ trên desktop và điện thoại; cận cảnh đào giun; cây mờ khi che; trận biểu diễn; thiếu gia ô tô và hai mẹ con |

## 22. Lộ trình

Ước lượng cho 1 người làm (ngày công), mỗi giai đoạn kết thúc bằng test xanh, ảnh chụp và cập nhật TIEN_DO.md.

| Giai đoạn | Nội dung | Ước lượng | Xong khi |
| --- | --- | --- | --- |
| **P0 – Chống kẹt và di chuyển** | Bộ `live`, độ no mới, quà 100 thóc, ăn sống, đào khi kiệt sức, chạy nhanh, cây mờ khi che | 2–3 | Người chơi mới không hết độ no trước 15 phút; không còn ngõ cụt; chạy nhanh đúng luật |
| **P1 – Đào giun v2** | Mổ, popup, cận cảnh, giằng co, máy chủ mô phỏng, XP khi đào, bù tỉ lệ | 3–4 | Test mô phỏng đạt tỉ lệ; e2e bắt/thoát |
| **P2 – Nền NPC và hướng dẫn** | Actor, `say`, bong bóng, thư viện thoại, lời chào, cài đặt, hướng dẫn 8 bước qua NPC | 3–4 | E2E người mới đi trọn |
| **P3 – Xe buýt và xe ôm** | Đường vành đai, bến, lịch xe, lượt hồi 5 phút, xe liên trại theo lịch, xe ôm | 4–5 | E2E đi xe; load test giữ tick |
| **P4 – Làng sống** | Giao thông NPC, cameo, trận biểu diễn, chủ quán chủ động, phản ứng theo trạng thái | 3–4 | Chụp ảnh và quay đủ các cảnh; hạn mức đúng |
| **P5 – Kho đồ, thưởng cấp, đất và nhà** | Items, bảng thưởng, Chủ đầu tư, lô, vật liệu, xây và nâng cấp; P5b: lô trưng bày ở Xóm | 4–5 | E2E mua đất và xây nhà |
| **P6 – Trồng cây và thú cưng có kỹ năng** | Hạt giống, luống nhà và luống cộng đồng, trứng nhiều loại, loài và kỹ năng | 3–4 | Test kỹ năng và giới hạn cộng dồn |
| **P7 – Bổ sung** | Điểm danh, ngày/đêm, thời tiết, nhiệm vụ NPC, bảng xếp hạng | 3–5 | Theo từng mục |

Tổng khoảng **25–34 ngày công**. P0 và P1 giải quyết ngay cảm giác "hết mana nhanh, kẹt, đi chậm, đào nhàm" nên làm trước. P2 là nền cho P3 và P4.

## 23. Rủi ro và điểm cần chủ dự án chốt

1. **Quà 100 thóc cho tài khoản cũ:** plan mặc định cấp một lần cho mọi tài khoản. Nếu chỉ muốn cho người mới, bỏ bước trong `normalize`.
2. **Mức trêu chọc:** mặc định bật, giọng nhẹ, có hạn mức và nút tắt. Nếu muốn an toàn hơn cho người nhỏ tuổi, đặt mặc định "Vừa" và tắt trêu.
3. **Giá vé vòng làng 3 thóc** và hồi lượt 5 phút có thể làm người chơi ít đi bộ. Theo dõi tỉ lệ đi bộ / xe buýt / xe ôm sau một tuần.
4. **Tỉ lệ giun thoát** là cảm giác "khó mà vui" hay "ức chế": chốt bằng chơi thử trước khi khoá số.
5. **Bot chơi giằng co hoàn hảo** được chấp nhận (mục 4.3); cần log để quản trị soi.
6. **Lô trưng bày ở Xóm** (P5b) phụ thuộc số người online mỗi kênh; nếu kênh vắng thì hiện nhà mẫu.
7. **Cấp tối đa:** giữ 10 ở bản này; mở tới 20 là một hạng mục riêng.
