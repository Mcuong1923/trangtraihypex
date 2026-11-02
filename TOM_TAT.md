# Trang trại gà Hypex: đã làm được gì

Cập nhật: 02/11/2026. Mã nguồn: https://github.com/Mcuong1923/trangtraihypex

Game nông trại 3D nhiều người chơi, chạy trên trình duyệt. Mỗi người điều khiển một chú gà: đi bộ khắp làng, đào giun, nấu ăn, mua đồ, đấu PK, kết bạn, xây nhà, lập hội. Máy chủ Go quyết định mọi luật chơi; trình duyệt (Three.js) chỉ gửi thao tác và hiển thị.

Chi tiết từng hạng mục và cách kiểm chứng: [TIEN_DO.md](TIEN_DO.md). Cách chạy: [README.md](README.md).

## Người chơi làm được gì

### Vào game
- Đăng ký, đăng nhập bằng tên + mật khẩu. Có mã khôi phục để lấy lại tài khoản.
- Tạo gà: 6 màu lông, 3 kiểu mào, 3 kiểu đuôi, 6 kiểu mắt, áo khởi điểm, giới tính hồ sơ. Xem trước 3D, kéo để xoay.
- Mất mạng hoặc tải lại trang thì có 15 giây để vào lại đúng chỗ cũ, kể cả khi đang đấu PK.

### Thế giới 3D
- **Sảnh chuồng gà:** 40 ổ rơm ngồi được, bục có Cụ Cục (chủ chuồng), máng ăn, máng nước, bảng tin. Có 8 NPC sinh hoạt; NPC đang ngồi ổ sẽ đứng dậy nhường khi người chơi lại gần.
- **Bản đồ liền, đi bộ thật, không dịch chuyển:** ra cửa sảnh là sân làng (khoảng 230 × 250 m). Đường đất từ quảng trường cây đa dẫn tới 9 khu, mỗi khu có cổng ghi tên: sảnh, vườn giun, chợ, sân tỷ thí, trạm ấp, ao, Xóm Rơm Vàng, bến xe, gara.
- **Trại thứ hai, Ven Suối:** đi xe buýt từ bến xe sang; có đồng ven suối và giun đặc sản riêng.
- **Camera:** kéo để xoay, cuộn chuột hoặc chụm hai ngón để zoom. Tường, cổng, nhà che gà sẽ tự mờ đi.
- **Di chuyển:** bàn phím, hoặc chạm/click xuống đất để gà tự tìm đường tới.
- **Bản đồ và chỉ đường:** bản đồ lớn vẽ đúng địa hình, có vị trí "BẠN" và người chơi khác. Minimap cuộn theo gà. "Đặt chỉ đường" vẽ tuyến theo đường làng, có thời gian đi bộ ước tính.
- Người cùng kênh thấy nhau ở mọi nơi trên bản đồ. Đổi kênh thì vẫn đứng nguyên chỗ cũ.
- Bị kẹt thì dùng "Về sảnh an toàn" trong Cài đặt.

### Vòng chơi chính
- Gà mới nhận **100 thóc khởi nghiệp**. Cụ Cục gọi đúng tên, dặn luật bằng bong bóng thoại rồi dẫn 6 việc đầu tiên (có thể bỏ qua).
- **Đào giun kiểu giằng co:** gà mổ đất, giun ló lên, camera kéo lại gần, giữ nút để kéo và thả khi dây căng đỏ. Giun hiếm kéo lâu hơn, tuột 3 lần là chạy mất. 5 loại giun ở vườn, Ven Suối có thêm 4 loại đặc sản. Túi chứa tối đa 60 con.
- Bán giun lấy thóc, chế biến rồi ăn, hoặc **ăn sống** ngay để hồi độ no.
- **Độ no** dùng được khoảng 15–20 phút. Không bao giờ kẹt: kiệt sức vẫn đào được giun thường, Cụ Cục cho một phần cám mỗi ngày khi trắng tay.
- **Chạy nhanh** (Shift hoặc nút 🏃): nhanh gấp 1,65 lần trong 6 giây, hụt hơi thì nghỉ 5 giây.
- Lên cấp bằng XP (đào cũng có XP). Cấp 2–10 tặng thóc, vật liệu, thức ăn, hạt giống, trứng. Điểm danh 7 ngày, nhiệm vụ ngày, sổ giun, kỷ lục và khoe giun.

### Thời trang
- 24 món đồ, cộng thêm 3 món chỉ bán trong mùa thu.
- Thử trên gà 3D trước khi mua; lưu được 3 bộ phối.

### Đấu PK
- 2 kiểu đấu: Mổ nhanh và Úp ngửa.
- Đấu với người hoặc Gà Máy. Có chặn auto-click.

### Làng sống
- **Xe buýt vòng làng** do NPC lái chạy liên tục qua 9 bến. Mỗi người có 2 lượt miễn phí, hồi 1 lượt mỗi 5 phút; hết lượt thì 3 thóc.
- **Xe ôm** chạy dọc đường, tự tạt vào mời. Đồng ý thì chọn điểm đến trên bản đồ, 1–2 thóc tuỳ xa gần.
- **NPC tự sinh hoạt:** xe đạp, xe máy, ván trượt trên đường; thiếu gia đi ô tô trêu người đi bộ; hai mẹ con gà đi ngang bình luận; shipper hỏi đường; nhà báo phỏng vấn khi đào được giun hiếm.
- **Trận biểu diễn:** hai gà NPC đấu ở Sân tỷ thí, Chó Mực làm trọng tài, khán giả cổ vũ, cá cược vui, cười nhạo. Có ngày hội thì trận dời ra sân khấu ở Ao và vườn.
- Chủ quán tự chào, ví dụ Dê Thợ Máy tặng phiếu giảm 10% xe đầu tiên. Chỉnh mức trò chuyện và tắt trêu trong Cài đặt.
- **Ngày và đêm:** một ngày trong làng dài 48 phút. Ban đêm đèn lồng sáng, Giun phát sáng ngoi lên nhiều hơn. Thỉnh thoảng có mưa: ụ đất hồi nhanh hơn, xe ôm phụ thu 1 thóc.
- Cây chắn tầm nhìn tự mờ đi.

### Bạn bè và cộng đồng
- Chat, 9 biểu cảm, bóng chat trên đầu gà.
- Kết bạn, tin nhắn riêng, chặn, báo cáo kèm bằng chứng.
- Bảng việc làng có thêm nhiệm vụ của NPC (thưởng vật liệu hoặc trứng), sổ khám phá, 10 danh hiệu, thư Chim Chíp, ngày hội góp thóc.
- **Bảng xếp hạng tuần:** giun dài nhất, nhiều giun hiếm nhất, thắng PK nhiều nhất. Hạng nhất nhận danh hiệu.
- Ngồi ổ cạnh bạn bè: bạn đồng hành của cả hai thân thiết hơn.
- Mở tiệc ở sân nhà (30 hoặc 60 phút) và chụp ảnh.
- Giao dịch hai bên (giun, thóc): cả hai khoá rồi xác nhận; sửa đề nghị thì phải khoá lại. Mở từ cấp 3.

### Nhà riêng
- **Ông Chủ Đầu Tư** ở cổng Xóm Rơm Vàng bán lô đất: 3 hạng, 3 vị trí (cạnh đường, ven ao, đồi thông).
- **Bò Mộc** bán vật liệu (rơm, gỗ, đinh, đá, gạch, ngói) và xây nhà theo thiết kế: chuồng nhỏ hoặc nhà hai gian, bằng rơm, gỗ hoặc gạch; nâng cấp chỉ trả phần chênh lệch. Vẫn tính giờ khi offline. Nhà càng chắc thì ngủ ổ trong nhà hồi độ no càng nhanh và chứa được nhiều khách hơn.
- **Trồng cây:** mua hạt ở Quầy thức ăn, gieo ở 12 luống cộng đồng (Ao và vườn) hoặc luống rau trong nhà, tưới cho nhanh lớn, thu trước khi héo. Bán hoặc ăn nông sản.
- Đặt và xoay 10 loại đồ nội thất.
- Đặt quyền vào nhà, mời và đuổi khách. Vào nhà người khác qua Xóm Rơm Vàng.
- **Hợp đồng thợ xây:** chủ nhà ký quỹ; thợ từ cấp 7 nhận việc. Máy chủ tự nghiệm thu và trả công; quá hạn thì tự hoàn tiền.

### Trứng và bạn đồng hành
- Trứng thường, trứng đốm, trứng vàng đến từ quà lên cấp, điểm danh và nhiệm vụ NPC. Ấp ở Trạm ấp của Vịt Bông (30 phút, vẫn chạy khi offline).
- Nở ra 7 loài, mỗi loài một kỹ năng: Gà con đánh hơi giun hiếm, Vịt con mặc cả giá vé, Ngỗng con cõng chạy (thân thiết nhất thì cưỡi ngỗng), Chim cút đi nhanh, Gà tre lâu đói, Gà ác bán giun giá cao, Công con có cơ hội đào được 2 con.
- Bạn đồng hành đi theo chủ trong cảnh 3D, chăm thì thân hơn, kỹ năng mạnh dần. Mang theo tối đa 3 con; xây nhà rồi thì nuôi thêm 6 con.

### Xe cộ
- **Xe buýt Chim Chíp** đi Ven Suối: 8 ghế, xuất bến theo lịch, dùng chung lượt miễn phí với xe buýt vòng làng; hết lượt thì 10 thóc. Huỷ vé được hoàn tiền. Về Hypex miễn phí.
- **Xe đạp, xe máy, ô tô:** mua ở cửa hàng, lấy và cất ở gara, chở bạn bè. Xe máy và ô tô chở cả nhóm sang trại khác.
- Không lái xe được vào sảnh, vườn đào, sân PK, trạm ấp và trong nhà; xe dừng ở mép khu.

### Hội Gà
- Lập hội từ cấp 10 với 1.000 thóc, tối đa 30 thành viên.
- Có chủ, phó và thành viên; duyệt đơn vào hội; chat hội.
- Quỹ hội chỉ dùng để xây công trình (cờ hội, đèn lồng, Nhà Hội cấp 2). Chủ hội vắng lâu thì phó hội được nhận quyền.
- Thi đua theo tuần. Nhà Hội riêng, chỉ thành viên vào được.

### Theo mùa
- Mùa thu: bộ đồ riêng, giun sự kiện "Giun bí ngô" và món "Bánh thóc mùa thu".

### Khác
- Âm thanh có chỉnh âm lượng. Đồ hoạ 3 mức thấp, vừa, cao. Có tuỳ chọn giảm chuyển động.
- Giao diện dùng được trên điện thoại.

## Quản trị và vận hành
- Trang `/admin.html` chia 3 vai trò: kiểm duyệt, kinh tế, admin.
- Hàng đợi báo cáo. Khoá chat hoặc khoá tài khoản có thời hạn và lý do.
- Gửi thông báo toàn hệ thống, bật chế độ bảo trì, tra sổ cái thóc, xem nhật ký quản trị.
- Xem số đo máy chủ: thời gian tick, bộ nhớ, số goroutine, số gói tin bị bỏ.
- Mở được cho máy khác trong cùng wifi (ví dụ `http://192.168.1.113:8080`).

## Kỹ thuật
- **Máy chủ:** Go, mô phỏng 20 lần/giây, gửi trạng thái cho trình duyệt 10 lần/giây qua WebSocket. Máy chủ tự tính vị trí và khu đang đứng; trình duyệt không gửi toạ độ hay kết quả.
- **Chống gian lận và mất tài sản:** mỗi thay đổi tài sản có mã yêu cầu nên không bị xử lý hai lần. Dữ liệu được ghi xuống đĩa trước khi báo thành công; ghi lỗi thì hoàn lại. Có sổ cái thóc.
- **Lưu trữ:** file JSON ghi nguyên tử. Phần lưu trữ đã tách riêng để sau này đổi sang PostgreSQL.
- **Trình duyệt:** Three.js r128 kèm sẵn trong repo. Mọi mô hình và chất liệu được dựng bằng code, không cần file 3D ngoài.
- **Ba bộ thông số:** `live` (mặc định, chơi thật theo [PLAN_LOGIC_GAME.md](PLAN_LOGIC_GAME.md)); `demo` rút ngắn thời gian để thử nhanh; `plan` giữ số gốc của PLAN.md để đối chiếu.
- **Đào giun chống gian lận:** máy chủ gieo loài giun và nhịp giằng co, trình duyệt gửi lại dòng thời gian giữ/thả, máy chủ chạy lại đúng mô phỏng đó (bản JS và Go được test khớp nhau từng bước).
- **Quy mô mã:** máy chủ khoảng 12.800 dòng, test khoảng 3.600 dòng, client khoảng 6.800 dòng JavaScript.

## Đã kiểm chứng
- **Test:** 69 test (có race detector). Gồm luật chơi và các test chơi thật qua WebSocket: đi bộ sảnh → vườn → chợ → sân PK, lái xe chở bạn sang bến, đi xe buýt và xe ôm, mua đất xây nhà, 0 thóc 0 độ no vẫn chơi tiếp, giun hiếm thoát, bỏ qua hướng dẫn.
- **Load test:**
  - 1.000 bot: 0 lỗi, 0 gói bị bỏ, mỗi tick (p95) 0,24 ms so với ngân sách 50 ms.
  - 1.500 bot cùng lúc: 0 lỗi.
  - Chạy 2 giờ với 1.000 bot: bộ nhớ không tăng dần. Máy Mac ngủ giữa chừng nên cần chạy lại trên máy không ngủ.
- **Giao diện:** chụp ảnh từng khu trong trình duyệt chạy ẩn; thử kéo xoay, zoom, chạm để đi, đi bộ giữa các khu, bản đồ và chỉ đường. Ảnh các tính năng logic mới (đào giằng co, xe ôm, đất nhà, ngày/đêm, mưa…): [docs/visual-qa/logic/](docs/visual-qa/logic/).

## Chưa làm hoặc chưa kiểm chứng
- **FPS trên điện thoại và máy có card đồ hoạ thật** chưa đo. Ở quảng trường hiện khoảng 900 lượt vẽ mỗi khung hình; máy yếu nên chọn đồ hoạ Thấp.
- **Cảm ứng trên điện thoại thật** chưa thử.
- **Chạy nhiều máy chủ cùng lúc, PostgreSQL, Redis:** chưa làm vì máy dev chưa có. Hiện chạy một tiến trình.
- **Đăng nhập bằng email hoặc Google:** chưa có.
- **Khu đầy:** chưa có hàng đợi; hiện báo lỗi và gợi ý đổi kênh.
- **Ảnh chụp trại Ven Suối** sau khi làm lại bản đồ chưa có (đã có test chạy qua).
- **Logic để sau:** tưới giúp luống của bạn bè, lô trưng bày nhà người chơi ở Xóm, đèn pha xe NPC ban đêm, câu cá, mở cấp 11–20.
- **Cân bằng** độ no, giá vé, tỉ lệ giun thoát mới chốt bằng mô phỏng và test; cần chơi thử thật để chỉnh.
