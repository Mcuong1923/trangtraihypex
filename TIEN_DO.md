# Tiến độ dự án Trang trại gà Hypex

## Rà soát logic và sửa lỗi (05/11/2026)

Chi tiết: [plans/reports/code-reviewer-261105-1412-game-logic-review.md](plans/reports/code-reviewer-261105-1412-game-logic-review.md). Hiện có 128 test, chạy `go test -count=1 -timeout 25m ./...` (~12 phút); máy Windows hiện chưa có cgo nên chưa chạy `-race`.

- **Lỗi nghiêm trọng C1–C5, H1–H5:** chặn ID tài khoản đi vòng đường dẫn, bán giun lạ làm sập server, đổi kênh né hồi đất, recover panic trong phòng, hai phiên một tài khoản, gà "ma" khi chuyển phòng, giới hạn tốc độ message, trần câu cá 60/ngày, giới hạn báo cáo.
- **Lỗi chặn tính năng R1, B1–B3, D1:** "Bán hết" một lệnh, giao dịch và xe ôm không còn tự huỷ, nút xuống xe/mời xuống, Úp ngửa có hạn chọn 10 giây.
- **Đồng bộ với plan:** chỉ Mổ nhanh với người thật tính thắng (bảng tuần, điểm hội, "Cao thủ PK"); gà con cũ có kỹ năng Mỏ Thính; sức chứa nhà rơm 8/gỗ 12/gạch 16; quà cấp khi kho đầy vào hàng chờ (Túi đồ → Kho); bỏ lệnh nâng đất cũ `home_upgrade`; giao dịch có vật liệu, hạt giống, nông sản; xe liên trại chạy theo lịch 3 phút; client dọn màn giằng co khi lượt đào/câu bị huỷ.
- **Mất tiền, mất đồ (M1, M2, M4–M6, M12–M14):** quà cấp khi lên cấp nhờ ăn, cây không mất khi đổi lô/xây nhà, hợp đồng/hội/đất/quỹ hội không còn "miễn phí" khi lưu lỗi, điểm danh kho đầy không mất chuỗi, lượt đào/câu lỗi vẫn tính hồi.
- **Chốt luật và sửa lỗi còn lại (06/11/2026):** chốt các luật bên dưới và đồng bộ code, plan, README; xe liên trại chạy lịch cố định 3 phút (demo 90 giây), cửa mở 20 giây trước giờ chạy; khu đông sảnh 50, khu khác 25; lời mời hội và chuyển chủ hết hạn sau 24 giờ; đăng nhập tính là có mặt.
- **Giới hạn người trong khu (06/11/2026):** đi tới cổng khu đã đủ người thì đứng vào hàng chờ, người tới trước vào trước, còn đứng gần cổng là còn giữ chỗ (đi xa, mất mạng, lên xe thì mất), tới lượt được báo và có 30 giây để vào; người giữ vé luôn vào được bến xe; xuống xe (xe người chơi, xe buýt, xe ôm) hoặc đổi kênh ngay trong khu đã đầy thì được đặt ra ngoài cổng khu, kèm thông báo lý do.
- **Đã chốt (06/11/2026):** luống rau nhà theo lô 2/4/6/8; câu cá tối đa 30 con/ngày, riêng 120 lượt đào; thân thiết từ bạn bè tối đa 60/ngày, tặng hoa +5; PK xếp hạng chỉ Mổ nhanh giữa người thật, bỏ trận chỉ tính đã chơi, thắng cùng đối thủ quá 5 trận/tuần không tính; kho 200 gồm nông sản và cá, quà đầy kho vào hàng chờ; khách nghỉ ổ trong nhà người khác như chủ; cấp tối đa 20. Đã làm: tưới giúp, lô trưng bày Xóm, đèn pha xe ban đêm, câu cá, cấp 11–20.

## Logic game v2 (02/11/2026, theo [PLAN_LOGIC_GAME.md](PLAN_LOGIC_GAME.md))

Đủ các giai đoạn P0–P7 của plan. Server mặc định chạy bộ cân bằng `live` (`go run .`); `-balance demo` để thử nhanh. Toàn bộ 69 test xanh với `-race` (~12 phút), gồm so khớp giằng co Go ↔ JS bằng Node. Ảnh kiểm tra trên trình duyệt: [docs/visual-qa/logic/](docs/visual-qa/logic/).

| Giai đoạn | Đã làm | Kiểm chứng |
| --- | --- | --- |
| P0 Chống kẹt, di chuyển | Bộ `live` (độ no ~15–20 phút), quà 100 thóc khởi nghiệp, ăn sống giun, Cụ Cục cho cám cứu trợ, kiệt sức vẫn đào giun thường; chạy nhanh ×1,65 trong 6 giây, hụt hơi nghỉ 5 giây; cây che tầm nhìn tự mờ | `TestLiveManaLastsFifteenToTwentyMinutes`, `TestStarterGiftOnce`, `TestRawEatAndRationNeverStuck`, `TestStaminaSprintAndLock`, e2e `TestZeroThocZeroManaStillPlayable` |
| P1 Đào giun v2 | Mổ đất → giun ló (popup) → cận cảnh → giữ/thả giằng co; giun hiếm kéo lâu, tuột 3 lần là thoát; máy chủ chạy lại mô phỏng để chốt, chống gieo lại; XP khi đào | `TestTugRatesByTier`, `TestTugDeterministicAndBounded`, `TestTugMatchesClient` (Node), e2e `TestRareWormEscapeNotCountedDaily`; ảnh 07–08 |
| P2 Nền NPC, hướng dẫn | Bong bóng thoại, Cụ Cục gọi đúng tên gà mới, dặn luật 4 trang có nút bỏ qua, hướng dẫn 6 việc có dấu "!" và mũi tên, tự trao thưởng; cài đặt NPC (Đầy đủ/Vừa/Ít, tắt trêu) | e2e `TestNPCWelcomeStarterAndCheckIn`, `TestTutorialSkipKeepsStarter`, `TestEndToEnd`; ảnh 06 |
| P3 Xe buýt, xe ôm | 3 xe buýt NPC chạy liên tục qua 9 bến; 2 lượt miễn phí hồi 5 phút, hết lượt 3 thóc; chip 🎫 trên HUD; xe ôm số lượng theo người ngoài trời, mời đi, chọn điểm trên bản đồ, 1–2 thóc | e2e `TestVillageBusTokensThenFare`, `TestXeOmCarriesForOneOrTwoThoc`; ảnh 09 |
| P4 Làng sống | Xe đạp, xe máy, ván trượt; thiếu gia ô tô, hai mẹ con gà, shipper hỏi đường, nhà báo, bán rong, cụ dưỡng sinh; trận biểu diễn có Chó Mực và khán giả (dời ra sân khấu ao khi có ngày hội); chủ quán chủ động; hạn mức trêu/cameo, ≤ 3 câu/giây | `TestDirectorTeaseAndCameoLimits`, `TestRoomSayBudget`, e2e `TestArenaShowStartsWhenSomeoneWatches`; ảnh 10, 13 |
| P5 Kho, thưởng cấp, đất, nhà | Kho 200 món, quà cấp 2–10, điểm danh 7 ngày; Ông Chủ Đầu Tư bán lô (cạnh đường/ven ao/đồi thông); Bò Mộc bán vật liệu, xây nhà theo thiết kế + rơm/gỗ/gạch, nâng cấp; ngủ ổ trong nhà hồi độ no | `TestLotsHouseDesignsAndUpgrade`, `TestLevelRewardsOnceAndEggMigration`, `TestCheckInStreak`, e2e `TestBuyLotAndBuildHouseE2E`; ảnh 01–02 |
| P6 Trồng cây, thú cưng | Hạt giống ở Quầy thức ăn, luống cộng đồng ở Ao và vườn + luống rau nhà, tưới, héo ra phân ủ; 3 loại trứng, 7 loài với 7 kỹ năng (may mắn, mặc cả, cõng chạy, cưỡi ngỗng…) | `TestGardenGrowWaterHarvestWilt`, `TestPetAbilities`, `TestDigLuckKeepsTableNormalised`; ảnh 03 |
| P7 Bổ sung | Ngày/đêm 48 phút (đèn lồng, Giun phát sáng ×1,5, xe về nhà ban đêm, chào theo buổi); mưa ~10% (hồi đất −30%, xe ôm +1 thóc); nhiệm vụ NPC trên bảng việc làng; bảng xếp hạng tuần trao danh hiệu; nghỉ ổ cạnh bạn bè +1 thân thiết/phút | `TestGameClockDayNightAndRain`, `TestNightGlowKeepsTableNormalised`, `TestRainXeOmSurcharge`, `TestNPCQuestsWellFormed`, `TestWeeklyBoardsAndTitles`, `TestNestBuddyAffinity`; ảnh 04–05, 11–12 |

Đã làm sau 02/11 (xem 06/11/2026 ở đầu file): lô trưng bày nhà người chơi ở Xóm (P5b); tưới giúp luống của bạn bè; câu cá; mở cấp 11–20; đèn pha xe NPC ban đêm. Luống cộng đồng là "mỗi người một lớp" (ai cũng thấy cây của mình, không tranh luống) thay vì giữ chỗ độc quyền.

## Giao diện theo bộ ảnh 2 (02/11/2026, theo [PLAN_GIAO_DIEN_BO_2.md](PLAN_GIAO_DIEN_BO_2.md))

Đủ 11 cảnh đã dựng lại theo bố cục ảnh trong `assets/illustrations/bo-2` (mái rơm có mặt trong, vách lưới, lối ván, quầy/NPC đúng chỗ, ánh sáng ấm, HUD gỗ–kem). Mức hoàn thành: **dựng đủ bố cục và vật chủ đạo**; chưa đạt độ chi tiết của ảnh render. Bảng từng cảnh và phần còn thiếu: mục “Trạng thái triển khai” trong plan. Ảnh đối chiếu: [docs/visual-qa/bo-2/](docs/visual-qa/bo-2/).

- Server: dời Lợn Ủn (vườn), Chó Mực, Vịt Bông, Chim Chíp, Dê Thợ Máy, Bò Mộc và biển xóm tới vị trí trong ảnh; vật cản mới cho lán vườn, quầy thợ, nhà chờ + xe buýt (cả bến Ven Suối), khung nhà Bò Mộc, chuồng Ven Suối; khu “park” đổi tên thành “Ao và vườn”. Thêm cờ `-devwarp` (tắt mặc định) chỉ để chụp ảnh đối chiếu.
- Chưa làm: bảng điểm 3D và hiệu ứng KO ở sân PK; đo FPS trên máy thật.

## Làm lại thế giới 3D (01/11/2026, theo [PLAN_THE_GIOI_3D.md](PLAN_THE_GIOI_3D.md))

| Hạng mục | Trạng thái | Kiểm chứng |
| --- | --- | --- |
| Sảnh 32 × 42, 40 ổ rơm đan, bục cao 1,2 có Cụ Cục ngồi ổ lớn, tường gỗ, máng ăn/nước, bảng tin, cửa “Chuồng gà Hypex” | ✅ | Ảnh chụp góc từ cửa, cạnh bục, khi ngồi ổ; so với ảnh mẫu 01 |
| 8 NPC trong sảnh (chủ chuồng + đón khách, quét sân, mê ngủ, 2 hàng xóm có đi uống nước, ham đọc, chim đưa thư); NPC ngồi ổ tự đứng sang lối đi khi người chơi lại gần | ✅ | Ảnh chụp; NPC không tính vào số người chơi |
| Camera: góc sảnh từ cửa ↔ theo gà, kéo xoay ngang + ngẩng/cúi, zoom chuột/hai ngón, đổi khu không quay 180°, vật cản (cổng, cửa, tường, nhà) mờ đi khi che gà | ✅ | Thử bằng sự kiện chuột thật trong Chromium headless; cảm ứng chưa thử trên máy thật |
| Chạm/click đất để đi (tìm đường quanh bục, máng); kéo hoặc pinch không phát lệnh đi | ✅ | Đã thử: kéo đổi góc, không đi; click thì gà đi tới |
| **Bản đồ liền, không dịch chuyển:** mỗi trại là một bản đồ lớn (Hypex ≈ 230 × 250 m). Ra cửa sảnh là sân làng; đường đất từ quảng trường cây đa dẫn tới 9 khu, mỗi khu có cổng vòm ghi tên. Khu hiện tại do máy chủ tính theo vị trí; bỏ toàn bộ cổng dịch chuyển (chỉ còn “Về sảnh an toàn”). Người cùng kênh thấy nhau ở mọi nơi | ✅ | `TestWorldTopology` + e2e đi bộ thật sảnh → vườn → chợ → sân PK, lái xe sang bến; ảnh chụp đi bộ sảnh → vườn, sảnh → bến xe |
| Đổi kênh giữ nguyên chỗ đang đứng; xe không chạy vào khu cấm lái (dừng ở mép); đi bộ rời bến tự huỷ vé chưa chạy | ✅ | e2e |
| Cảnh riêng cho vườn giun, chợ (3 sạp mái vải), sân tỷ thí, trạm ấp, ao & hội chợ, Xóm Rơm Vàng, bến xe, gara, bến và đồng Ven Suối | ✅ | Ảnh chụp từng khu |
| Bản đồ làng/các trang trại, NPC gặp được, thời gian đi bộ ước tính, “Đặt chỉ đường”; chấm vàng dưới đất, mũi tên ở chân gà, tuyến trên minimap, bám đường đất trong làng | ✅ | Ảnh chụp sảnh → làng → vườn |
| Vào nhà riêng / Nhà Hội chỉ từ Xóm Rơm Vàng (máy chủ kiểm tra); lời mời ở xa thành chỉ đường; rời nhà ra cổng xóm | ✅ | e2e cập nhật |
| “Về sảnh an toàn” chuyển vào Cài đặt, có xác nhận | ✅ | |
| Giới hạn sảnh 50, khu khác 25; kênh 50 | ✅ máy chủ | Khu đầy: hàng chờ ở cổng (tới trước vào trước, đứng gần cổng là giữ chỗ, tới lượt có 30 giây để vào; người giữ vé luôn vào được bến); xuống xe/đổi kênh trong khu đầy thì ra ngoài cổng. Kênh đầy vẫn báo lỗi và đề xuất kênh khác, chưa có hàng chờ kênh |
| Hiệu năng | ⚠️ một phần | Bản đồ liền: khu xa > 90 m ẩn sau sương mù, cảnh ngoài trời gộp theo ô 60 m để camera bỏ qua phần ngoài tầm nhìn. Đứng ở quảng trường ≈ 900 draw call, trong sảnh ≈ 830 (đo bằng Chromium chạy CPU). **Chưa đo FPS trên điện thoại/desktop có GPU thật** (A08/A09); máy yếu nên chọn đồ hoạ Thấp |

Các mục “✅ Đã hoàn thành” bên dưới vẫn đúng về luật chơi; cách tới từng địa điểm nay theo bảng trên.

Cập nhật: 31/10/2026, tối. So với [PLAN.md](PLAN.md) bản 0.3.

## Tóm tắt

| Giai đoạn | Máy chủ (Go) | Giao diện (Three.js) | Ghi chú |
| --- | --- | --- | --- |
| **V1 công khai** | ✅ Xong | ✅ Xong | Còn hạ tầng production (xem mục ❌) |
| **V1.1** | ✅ Xong + có test | ✅ Xong, đã kiểm bằng ảnh chụp | Ấp trứng, bến xe & Ven Suối, xe đạp, giao dịch, hợp đồng thợ |
| **V1.2** | ✅ Xong + có test | ✅ Xong | Xe máy/ô tô liên trại, Hội Gà + Nhà Hội, nội dung theo mùa |

Kiểm thử:
- **40/40 test pass có race detector** (01/11/2026, bản đồ liền; khoảng 9,3 phút vì test end-to-end đi bộ thật trên bản đồ rộng). Chạy với `-timeout 20m`, vì giới hạn mặc định 10 phút của `go test` sát với thời gian chạy. Gồm unit test luật chơi và test end-to-end qua WebSocket thật.
- **Load test 1.000 bot** (khoảng 4 phút): 0 lỗi, 0 gói bị bỏ, tick p95 0,24 ms trên ngân sách 50 ms, heap khoảng 60 MB.
- **Tăng đột biến 1.500 bot:** 0 lỗi, tick p95 0,16 ms.
- **Soak 2 giờ với 1.000 bot** (2 giờ tính theo thời gian máy thức):
  - 1.000/1.000 bot giữ kết nối, 0 lỗi, 0 gói bị bỏ.
  - Tick p95 0,21 ms; RTT p95 0,9 ms.
  - Heap 45–76 MB không tăng dần, goroutine cố định 2.174: không thấy rò rỉ.
  - Khoảng 13.000 tin/giây, 72.293 trận PK, 414.107 hành động.
  - **Hạn chế:** máy Mac ngủ trong đêm nên bài bị gián đoạn, không phải 2 giờ liên tục. Các tick vọt (tối đa ~347 ms) và RTT tối đa ~944 giây trùng lúc máy ngủ dậy hoặc lúc chạy bộ test song song. Nên chạy lại trên máy chủ không ngủ trước khi công bố sức chứa.
- **Kiểm tra giao diện:** chơi tự động trong WebKit chạy ẩn (cùng engine với Safari), có chụp ảnh từng khu. **Chưa** thử trên trình duyệt có cửa sổ thật; bản local đang mở để chủ dự án tự thử.

Quy mô mã: server khoảng 7.900 dòng, test khoảng 2.000 dòng, client khoảng 3.900 dòng.

Cách chạy:
- Chơi: `cd server && go run .`, rồi mở `http://localhost:8080`.
- Quản trị: chạy kèm `-staff tenban:admin`, rồi mở `/admin.html`.
- Chi tiết: xem [README.md](README.md).

---

## ✅ Đã hoàn thành (server + giao diện)

### V1: lõi game
- **Tài khoản (mục 5, 14):**
  - Đăng ký/đăng nhập; mật khẩu băm PBKDF2; tạm chặn sau 8 lần nhập sai.
  - Mã khôi phục chỉ hiện một lần; đặt lại mật khẩu thì huỷ mọi phiên cũ.
- **Tạo gà:** 6 màu lông, 3 mào, 3 đuôi, 6 kiểu mắt, áo khởi điểm, giới tính hồ sơ; tên có kiểm duyệt; mỗi người có mã người chơi riêng.
- **Nhiều phòng (mục 6.3):**
  - Chuồng công cộng tối đa 25 người. Phiên mới luôn vào sảnh. Mỗi tài khoản chỉ một phiên điều khiển.
  - Đổi chuồng, mời bạn vào chuồng.
  - **Nối lại trong 15 giây**, kể cả giữa trận PK hay khi tải lại trang.
- **Thế giới 3D (theo ảnh 1–5):** sảnh, vườn giun, chợ, sân tỷ thí, chuồng riêng; camera nhìn chéo từ trên như ảnh.
- **Vòng chơi chính (mục 7, 8):**
  - Đào 5 loại giun đúng bảng 7.2; túi tối đa 60 giun.
  - Bán, hoặc chế biến & ăn (nguyên tử: thiếu thóc thì không mất giun).
  - Mana/đói có phạt; việc cứu đói khi hết thóc.
  - XP và cấp; hướng dẫn 15 phút đầu; nhiệm vụ ngày.
  - Khoe giun, sổ giun và kỷ lục.
- **Thời trang (mục 8.3):** 24 món + 3 món mùa thu; thử trên gà 3D trước khi mua; 3 bộ phối.
- **PK (mục 10):** Mổ nhanh và Úp ngửa, đấu người hoặc Gà Máy; chặn auto-click.
- **Xã hội & cộng đồng (mục 11):**
  - Chat chuồng, 9 biểu cảm, bạn bè, tin riêng xuyên phòng, chặn, báo cáo kèm bằng chứng.
  - Bảng việc làng, sổ khám phá + 6 danh hiệu, thư Chim Chíp, ngày hội góp thóc.
  - **Bữa tiệc sân nhà** (30/60 phút, ăn mừng ở bàn tiệc) và **chụp ảnh**.
- **Nhà riêng (mục 9.1):** 3 hạng đất, Bò Mộc xây chuồng (tính giờ cả khi offline), đặt và xoay 10 loại đồ, quyền vào nhà, mời và đuổi khách.
- **Vận hành (mục 18, 19):**
  - Trang quản trị phân vai: kiểm duyệt / kinh tế / admin.
  - Hàng đợi báo cáo; khoá chat/tài khoản có thời hạn và lý do.
  - Thông báo toàn hệ thống, chế độ bảo trì, tra sổ cái, nhật ký quản trị.
  - Số đo vận hành: tick, bộ nhớ, goroutine, gói bị bỏ.
- **Khác:**
  - Âm thanh có chỉnh âm lượng.
  - Cài đặt đồ hoạ thấp/vừa/cao, giảm chuyển động.

### V1.1
- **Trạm ấp trứng & bạn đồng hành (mục 12, ảnh 6):**
  - Vịt Bông tặng trứng ở cấp 5/7/9; ấp 2 giờ, vẫn chạy khi offline.
  - Nở đúng 1 con, đặt tên được; gà con đi theo chủ trong cảnh 3D.
  - Chăm sóc tăng độ thân thiết; tối đa 3 con.
- **Bến xe Chim Chíp & trại Ven Suối (mục 8.5, ảnh 9, 11):**
  - Vùng thứ hai có 4 loại giun đặc sản.
  - Xe liên trại 8 ghế, chạy theo lịch mỗi 3 phút (cửa mở 20 giây trước giờ chạy); dùng chung 2 lượt miễn phí hồi 5 phút, hết thì 10 thóc; huỷ vé hoàn đúng một lần.
  - Về Hypex miễn phí; đối soát chuyến dở khi máy chủ lỗi.
- **Xe đạp chở bạn (mục 8.4, ảnh 10):**
  - Cửa hàng xe, lấy/cất ở điểm đỗ, mời bạn lên ghế.
  - Cấm lái ở sảnh, khu đào, nhà và sân PK.
- **Giao dịch hai bên (mục 8.2):** giun và thóc; cả hai khoá rồi xác nhận; sửa đề nghị thì phải khoá lại; từ cấp 3.
- **Hợp đồng thợ xây (mục 9.2):**
  - Chủ ký quỹ; thợ từ cấp 7 nhận việc và vào nhà thi công.
  - Máy chủ tự nghiệm thu và trả công; quá hạn tự hoàn.
  - Uy tín giới hạn theo cặp chủ–thợ mỗi tuần.

### V1.2
- **Xe máy, ô tô (mục 8.4):** chở nhóm qua đường liên trại; cả nhóm vào cùng một bản sao chuồng, ghế giữ nguyên.
- **Hội Gà (mục 13):**
  - Lập hội từ cấp 10 với 1.000 thóc, tối đa 30 người.
  - Vai trò chủ/phó/thành viên, duyệt đơn, chat hội xuyên phòng.
  - Quỹ chỉ chi cho công trình (cờ hội, đèn lồng, Nhà Hội cấp 2), không rút về ví.
  - Chuyển chủ cần xác nhận; phó hội nhận quyền khi chủ vắng lâu; thi đua tuần.
  - **Nhà Hội** theo ảnh 7: chỉ thành viên vào, công trình đã xây hiện lên sân.
- **Nội dung theo mùa (mục 15):**
  - Bộ sưu tập mùa thu chỉ bán trong mùa.
  - Giun sự kiện "Giun bí ngô" 0,5%, các loài khác co lại theo tỉ lệ nên tổng vẫn bằng 1.
  - Món ăn mới "Bánh thóc mùa thu".

### Kỹ thuật cốt lõi (mục 16, 17)
- Máy chủ quyết định mọi thứ: client chỉ gửi hướng đi và ý định.
- Mọi giao dịch tài sản có request ID chống xử lý lặp, lưu xuống đĩa trước khi báo thành công, lỗi thì hoàn nguyên.
- Có sổ cái thóc.

---

## ❌ Chưa làm

| Hạng mục | Mục plan | Lý do |
| --- | --- | --- |
| Chạy nhiều tiến trình máy chủ | 18.3 | Cần kho dữ liệu dùng chung (PostgreSQL) và kênh tin xuyên máy (Redis). Máy này không có PostgreSQL, Redis hay Docker nên chưa dựng và kiểm chứng được. Hiện chạy 1 tiến trình |
| PostgreSQL + Redis | 16.1 | Như trên. Đang lưu file JSON ghi nguyên tử; giao diện `Store` đã tách để thay sau |
| Đăng nhập email / OAuth | 14 | Cần nhà cung cấp bên ngoài (Google, máy chủ gửi mail), môi trường này không có. Hiện có tên đăng nhập + mật khẩu + mã khôi phục |
| Đo FPS trên điện thoại thật | 18.2 | Cần thiết bị Android/iPhone thật |
| Thử trên trình duyệt có cửa sổ | — | Chủ dự án đang tự thử bản local |

## Khác với plan

- **Ngôn ngữ server:** Go, theo kiến trúc chủ dự án chọn, thay cho Node.js + Colyseus.
- **Lưu trữ:** file JSON thay cho PostgreSQL (xem bảng trên).
- **Snapshot xe NPC:** vẫn gửi vị trí mọi xe NPC trong phòng (bản đồ và minimap cần xe buýt), không lọc bán kính 70 m như PLAN_LOGIC_GAME 5.1.
- **Xe ôm:** không có đường máy chủ huỷ chuyến đang chở. Không hoàn tiền khi người chơi tự xuống hoặc mất kết nối là đúng plan.

## Việc tiếp theo

1. Sửa các lỗi chủ dự án báo lại sau khi tự chơi thử (bản local đang chạy ở `http://localhost:8080`).
2. Chạy lại soak 2 giờ liên tục trên máy không ngủ.
3. Khi chuẩn bị lên production: chuyển sang PostgreSQL + Redis, chạy nhiều instance, đo trên điện thoại thật.
